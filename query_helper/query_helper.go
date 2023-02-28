package queryhelper

import (
	pg_query "github.com/pganalyze/pg_query_go/v4"

	"github.com/mhkarimi1383/pg_pro/types"
)

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
func GetRelatedTables(q string) (tables []types.TableAccessInfo, err error) {
	result, err := pg_query.Parse(q)
	if err != nil {
		return
	}

	for _, i := range result.Stmts {
		if selectStmt := i.Stmt.GetSelectStmt(); selectStmt != nil {
			for _, from := range selectStmt.GetFromClause() {
				if from.GetRangeVar() != nil {
					tables = append(tables, types.TableAccessInfo{
						TableInfo: types.TableInfo{
							Name:   from.GetRangeVar().Relname,
							Schema: types.SchemaNameFixer(from.GetRangeVar().Schemaname),
						},
						AccessMode: types.Select,
					})
				} else {
					tables = append(tables, types.TableAccessInfo{
						AccessMode: types.Select,
					})
				}
			}
		} else if insertStmt := i.Stmt.GetInsertStmt(); insertStmt != nil {
			tables = append(tables, types.TableAccessInfo{
				TableInfo: types.TableInfo{
					Name:   insertStmt.Relation.Relname,
					Schema: types.SchemaNameFixer(insertStmt.Relation.Schemaname),
				},
				AccessMode: types.Insert,
			})
		} else if deleteStmt := i.Stmt.GetDeleteStmt(); deleteStmt != nil {
			tables = append(tables, types.TableAccessInfo{
				TableInfo: types.TableInfo{
					Name:   deleteStmt.Relation.Relname,
					Schema: types.SchemaNameFixer(deleteStmt.Relation.Schemaname),
				},
				AccessMode: types.Delete,
			})
		} else if updateStmt := i.Stmt.GetUpdateStmt(); updateStmt != nil {
			tables = append(tables, types.TableAccessInfo{
				TableInfo: types.TableInfo{
					Name:   deleteStmt.Relation.Relname,
					Schema: types.SchemaNameFixer(updateStmt.Relation.Schemaname),
				},
				AccessMode: types.Update,
			})
		} else {
			tables = append(tables, types.TableAccessInfo{
				TableInfo: types.TableInfo{
					Name:   "",
					Schema: "",
				},
				AccessMode: types.System,
			})
		}
	}
	return
}
