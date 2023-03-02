package types

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgproto3"
)

type QueryResult struct {
	pgproto3.RowDescription
	DataRows []pgproto3.DataRow
}

type TableAccessMode int64

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

func (tam TableAccessMode) ToString() string {
	return []string{"SELECT", "DELETE", "INSERT", "UPDATE", "SYSTEM", "INVALID"}[tam]
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

func SchemaNameFixer(name string) string {
	if name == "" {
		return defaultSchemaName
	}
	return name
}
