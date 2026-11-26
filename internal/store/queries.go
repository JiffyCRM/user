package store

import (
	"embed"
	"fmt"

	"github.com/JiffyCRM/user/internal/sqlquery"
)

//go:embed sql/*.sql
var queryFiles embed.FS

var queries = mustLoadQueries()

func mustLoadQueries() *sqlquery.Set {
	set, err := sqlquery.Load(queryFiles, "sql/*.sql")
	if err != nil {
		panic(fmt.Sprintf("load embedded store queries: %v", err))
	}
	return set
}

func query(name string) string {
	return queries.MustGet(name)
}