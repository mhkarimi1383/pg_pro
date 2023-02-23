package queryhelper

import (
	"fmt"

	pg_query "github.com/pganalyze/pg_query_go/v4"
)

const (
	Select TableAccessMode = iota
	Delete
	Insert
	Update
	System
	Invalid
)

const (
	defaultSchemaName = "public"
)

type TableAccessMode int64

func (tam TableAccessMode) ToString(index TableAccessMode) string {
	return []string{"SELECT", "DELETE", "INSERT", "UPDATE", "SYSTEM", "INVALID"}[index]
}

func TableAccessModeFromString(s string) (index TableAccessMode, err error) {
	switch s {
	case "SELECT":
		index = Select
	case "DELETE":
		index = Delete
	case "INSERT":
		index = Insert
	case "UPDATE":
		index = Update
	case "SYSYEM":
		index = System
	default:
		return Invalid, fmt.Errorf("invalid input %v", s)
	}
	return
}

type TableInfo struct {
	Name   string
	Schema string
}

type TableAccessInfo struct {
	TableInfo
	AccessMode TableAccessMode
}

func schemaNameFixer(name string) string {
	if name == "" {
		return defaultSchemaName
	}
	return name
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
				if from.GetRangeVar() != nil {
					tables = append(tables, TableAccessInfo{
						TableInfo: TableInfo{
							Name:   from.GetRangeVar().Relname,
							Schema: schemaNameFixer(from.GetRangeVar().Schemaname),
						},
						AccessMode: Select,
					})
				} else {
					tables = append(tables, TableAccessInfo{
						AccessMode: Select,
					})
				}
			}
		} else if insertStmt := i.Stmt.GetInsertStmt(); insertStmt != nil {
			tables = append(tables, TableAccessInfo{
				TableInfo: TableInfo{
					Name:   insertStmt.Relation.Relname,
					Schema: schemaNameFixer(insertStmt.Relation.Schemaname),
				},
				AccessMode: Insert,
			})
		} else if deleteStmt := i.Stmt.GetDeleteStmt(); deleteStmt != nil {
			tables = append(tables, TableAccessInfo{
				TableInfo: TableInfo{
					Name:   deleteStmt.Relation.Relname,
					Schema: schemaNameFixer(deleteStmt.Relation.Schemaname),
				},
				AccessMode: Delete,
			})
		} else if updateStmt := i.Stmt.GetUpdateStmt(); updateStmt != nil {
			tables = append(tables, TableAccessInfo{
				TableInfo: TableInfo{
					Name:   deleteStmt.Relation.Relname,
					Schema: schemaNameFixer(updateStmt.Relation.Schemaname),
				},
				AccessMode: Update,
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
