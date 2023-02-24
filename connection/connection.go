package connection

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/mhkarimi1383/pg_pro/cache"
	"github.com/mhkarimi1383/pg_pro/config"
	"github.com/mhkarimi1383/pg_pro/logger"
	"github.com/mhkarimi1383/pg_pro/types"
	"github.com/mhkarimi1383/pg_pro/utils"
)

var (
	writePools []*pgxpool.Pool
	readPools  []*pgxpool.Pool
)

func init() {
	for _, source := range config.GetSlice("sources") {
		src := source.(map[string]any)
		cfg, err := pgxpool.ParseConfig(fmt.Sprintf("%v", src["url"]))
		if err != nil {
			panic(errors.Wrap(err, "pgxpool config parse"))
		}

		minConns, err := strconv.Atoi(fmt.Sprintf("%v", src["min_conns"]))
		if err != nil {
			panic(errors.Wrap(err, "converting min_conns to number"))
		}
		cfg.MinConns = int32(minConns)
		maxConns, err := strconv.Atoi(fmt.Sprintf("%v", src["max_conns"]))
		if err != nil {
			panic(errors.Wrap(err, "converting max_conns to number"))
		}
		cfg.MaxConns = int32(maxConns)
		pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
		if err == nil {
			if src["mode"] == "slave" {
				readPools = append(readPools, pool)
			} else if src["mode"] == "master" {
				writePools = append(writePools, pool)
			}
		}
	}
	if len(writePools) > 1 {
		panic("multiple write connections provided")
	}
}

func RunQuery(q string, readOperation bool) (result *types.QueryResult, err error) {
	defer func() {
		if err == nil && readOperation {
			cacheSetErr := cache.Set(q, result)
			if cacheSetErr != nil {
				logger.Warn(cacheSetErr.Error(), zap.String("event", "cache_set"))
			}
		}
	}()
	result = new(types.QueryResult)
	cacheResult, err := cache.Get(q)
	if readOperation && err == nil && cacheResult != nil {
		result = cacheResult
		return
	}
	if err != nil {
		logger.Warn(errors.Wrap(err, "error while reading cached data, using postgresql itself").Error(), zap.String("event", "cache_read"))
		err = nil
	}
	var pool *pgxpool.Pool
	if readOperation && len(readPools) > 0 {
		pool = readPools[rand.Intn(len(readPools))]
	} else {
		pool = writePools[rand.Intn(len(writePools))]
	}
	rows, err := pool.Query(context.Background(), q)
	if err != nil {
		return
	}
	for _, desc := range rows.FieldDescriptions() {
		result.RowDescription.Fields = append(result.RowDescription.Fields, pgproto3.FieldDescription{
			Name:                 []byte(desc.Name),
			Format:               desc.Format,
			TypeModifier:         desc.TypeModifier,
			DataTypeSize:         desc.DataTypeSize,
			DataTypeOID:          desc.DataTypeOID,
			TableOID:             desc.TableOID,
			TableAttributeNumber: desc.TableAttributeNumber,
		})
	}
	for rows.Next() {
		values, rowValuesErr := rows.Values()
		if rowValuesErr != nil {
			return nil, rowValuesErr
		}
		dataRow := pgproto3.DataRow{}
		for _, value := range values {
			byteValue := utils.GetBytes(value)
			dataRow.Values = append(dataRow.Values, byteValue)
		}
		result.DataRows = append(result.DataRows, dataRow)
	}
	return
}
