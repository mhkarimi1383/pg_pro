package connection

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	"github.com/mhkarimi1383/pg_pro/config"
	"github.com/mhkarimi1383/pg_pro/utils"
)

type QueryResult struct {
	pgproto3.RowDescription
	DataRows []pgproto3.DataRow
}

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

func RunQuery(q string, master bool) (result *QueryResult, err error) {
	result = new(QueryResult)
	var pool *pgxpool.Pool
	if master || len(readPools) == 0 {
		pool = writePools[rand.Intn(len(writePools))]
	} else {
		pool = readPools[rand.Intn(len(readPools))]
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
