package connection

import (
	"log"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgproto3"

	"github.com/mhkarimi1383/pg_pro/utils"
)

type QueryResult struct {
	pgproto3.RowDescription
	DataRows []pgproto3.DataRow
}

func RunQuery(q string) (result *QueryResult, err error) {
	result = new(QueryResult)
	conn, err := pgx.Connect(pgx.ConnConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "postgres",
		User:     "postgres",
		Password: "postgres",
	})
	if err != nil {
		return
	}
	defer conn.Close()
	rows, err := conn.Query(q)
	if err != nil {
		return
	}
	for _, desc := range rows.FieldDescriptions() {
		dataTypeOID, dataTypeOIDErr := desc.DataType.Value()
		if dataTypeOIDErr != nil {
			return nil, dataTypeOIDErr
		}
		tableOID, tableOIDOIDErr := desc.DataType.Value()
		if tableOIDOIDErr != nil {
			return nil, tableOIDOIDErr
		}
		result.RowDescription.Fields = append(result.RowDescription.Fields, pgproto3.FieldDescription{
			Name:                 []byte(desc.Name),
			Format:               desc.FormatCode,
			TypeModifier:         desc.Modifier,
			DataTypeSize:         desc.DataTypeSize,
			DataTypeOID:          uint32(dataTypeOID.(int64)),
			TableOID:             uint32(tableOID.(int64)),
			TableAttributeNumber: desc.AttributeNumber,
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
	log.Println("Values", len(result.DataRows))
	log.Println("Fields", len(result.RowDescription.Fields))
	return
}
