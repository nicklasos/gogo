package testkit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Row is the columns a row must have, for the database assertions:
//
//	app.AssertDatabaseHas("examples", testkit.Row{"user_id": user.ID, "title": "Hello"})
//
// A nil value matches NULL.
type Row map[string]any

func (a *App) AssertDatabaseHas(table string, row Row) {
	a.t.Helper()
	if a.count(table, row) == 0 {
		a.t.Fatalf("expected a row in %s matching %v", table, row)
	}
}

func (a *App) AssertDatabaseMissing(table string, row Row) {
	a.t.Helper()
	if count := a.count(table, row); count != 0 {
		a.t.Fatalf("expected no row in %s matching %v, found %d", table, row, count)
	}
}

// AssertDatabaseCount checks how many rows match. Pass no row to count the whole table.
func (a *App) AssertDatabaseCount(table string, want int, row ...Row) {
	a.t.Helper()
	var where Row
	if len(row) > 0 {
		where = row[0]
	}
	if count := a.count(table, where); count != want {
		a.t.Fatalf("expected %d rows in %s matching %v, found %d", want, table, where, count)
	}
}

func (a *App) count(table string, row Row) int {
	a.t.Helper()

	columns := make([]string, 0, len(row))
	for column := range row {
		columns = append(columns, column)
	}
	sort.Strings(columns)

	var conditions []string
	var args []any
	for _, column := range columns {
		name := pgx.Identifier{column}.Sanitize()
		if row[column] == nil {
			conditions = append(conditions, name+" IS NULL")
			continue
		}
		args = append(args, row[column])
		conditions = append(conditions, fmt.Sprintf("%s = $%d", name, len(args)))
	}

	query := "SELECT count(*) FROM " + pgx.Identifier{table}.Sanitize()
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := a.Tx.QueryRow(a.Ctx, query, args...).Scan(&count); err != nil {
		a.t.Fatalf("testkit: count rows in %s: %v", table, err)
	}
	return count
}
