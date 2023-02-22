package connection

import (
	"context"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mhkarimi1383/pg_pro/config"
	"github.com/mhkarimi1383/pg_pro/utils"
)

type QueryResult struct {
	pgproto3.RowDescription
	DataRows []pgproto3.DataRow
}

var (
	pool *pgxpool.Pool
)

func init() {
	var err error
	cfg, err := pgxpool.ParseConfig(config.GetString("sources.0.url"))
	if err != nil {
		panic(err)
	}

	cfg.MinConns = config.GetInt32("sources.0.min_conns")
	cfg.MaxConns = config.GetInt32("sources.0.max_conns")

	pool, err = pgxpool.NewWithConfig(context.Background(), cfg)

	if err != nil {
		panic(err)
	}
}

func RunQuery(q string) (result *QueryResult, err error) {
	result = new(QueryResult)
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
