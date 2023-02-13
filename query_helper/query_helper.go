package queryhelper

import (
	pg_query "github.com/pganalyze/pg_query_go/v2"
)

type TableAccessMode int64

func (tam TableAccessMode) ToString(index TableAccessMode) string {
	return []string{"SELECT", "DELETE", "INSERT", "UPDATE", "SYSTEM"}[index]
}

const (
	Select TableAccessMode = iota
	Delete
	Insert
	Update
	System
)

type TableInfo struct {
	Name   string
	Schema string
}

type TableAccessInfo struct {
	TableInfo
	AccessMode TableAccessMode
}

// IsReadOperation is used for caching data and master/replica load-balancing
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

// GetRelatedTables mostly used for access checking
func GetRelatedTables(q string) (tables []TableAccessInfo, err error) {
	result, err := pg_query.Parse(q)
	if err != nil {
		return
	}

	for _, i := range result.Stmts {
		if selectStmt := i.Stmt.GetSelectStmt(); selectStmt != nil {
			for _, from := range selectStmt.GetFromClause() {
				schema := from.GetRangeVar().Schemaname
				if schema == "" {
					schema = "public"
				}
				tables = append(tables, TableAccessInfo{
					TableInfo: TableInfo{
						Name:   from.GetRangeVar().Relname,
						Schema: schema,
					},
					AccessMode: Select,
				})
			}
		} else if insertStmt := i.Stmt.GetInsertStmt(); insertStmt != nil {
			schema := insertStmt.Relation.Schemaname
			if schema == "" {
				schema = "public"
			}
			tables = append(tables, TableAccessInfo{
				TableInfo: TableInfo{
					Name:   insertStmt.Relation.Relname,
					Schema: schema,
				},
				AccessMode: Insert,
			})
		} else if deleteStmt := i.Stmt.GetDeleteStmt(); deleteStmt != nil {
			schema := deleteStmt.Relation.Schemaname
			if schema == "" {
				schema = "public"
			}
			tables = append(tables, TableAccessInfo{
				TableInfo: TableInfo{
					Name:   deleteStmt.Relation.Relname,
					Schema: schema,
				},
				AccessMode: Delete,
			})
		} else if updateStmt := i.Stmt.GetUpdateStmt(); updateStmt != nil {
			schema := deleteStmt.Relation.Schemaname
			if schema == "" {
				schema = "public"
			}
			tables = append(tables, TableAccessInfo{
				TableInfo: TableInfo{
					Name:   deleteStmt.Relation.Relname,
					Schema: schema,
				},
				AccessMode: Delete,
			})
		} else {
			tables = append(tables, TableAccessInfo{
				TableInfo: TableInfo{
					Name:   "",
					Schema: "",
				},
				AccessMode: System,
			})
		}
	}
	return
}
