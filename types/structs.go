package types

import "github.com/jackc/pgx/v5/pgproto3"

type QueryResult struct {
	pgproto3.RowDescription
	DataRows []pgproto3.DataRow
}
