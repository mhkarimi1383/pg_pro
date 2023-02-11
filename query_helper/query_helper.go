package queryhelper

import (
	pg_query "github.com/pganalyze/pg_query_go/v2"
)

func IsReadOperation(q string) (isRead bool, err error) {
	result, err := pg_query.Parse(q)
	if err != nil {
		return
	}

	isRead = true
	for _, i := range result.Stmts {
		selectStmt := i.Stmt.GetSelectStmt()
		if selectStmt == nil {
			isRead = false
			break
		}
	}
	return
}
