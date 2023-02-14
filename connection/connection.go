package connection

import (
	"fmt"
	"os"

	"github.com/jackc/pgx"
)

func RunQuery(q string) {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	conn, err := pgx.Connect(pgx.ConnConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "postgres",
		User:     "postgres",
		Password: "postgres",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	rows, err := conn.Query(q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(rows.FieldDescriptions())
	for rows.Next() {
		v, _ := rows.Values()
		fmt.Println(v)
		if len(v) == 0 {
			os.Exit(1)
		}
	}
}

// func main() {
// 	RunQuery("SELECT 1;")
// }
