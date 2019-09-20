package querybuilder

import "testing"

func assertf(t *testing.T, ok bool, msg string, args ...interface{}) {
	t.Helper()
	if !ok {
		t.Errorf(msg, args...)
	}
}

func TestSelect(t *testing.T) {
	for _, tt := range []struct {
		stmt  *SelectStmt
		query string
		args  []interface{}
	}{{
		stmt:  Select(Call("unnest", Index("logs", Bind(1)))),
		query: "SELECT unnest(logs[$1:])",
		args:  []interface{}{1},
	}, {
		stmt: Select(
			As(Call("unnest", Index("logs", Bind(1))), "log"),
		),
		query: "SELECT unnest(logs[$1:]) AS log",
		args:  []interface{}{1},
	}, {
		stmt:  Select("foo").Where("bar"),
		query: "SELECT foo WHERE bar",
	}, {
		stmt: Select("col").
			From("table").
			Where(Expr(Expr(
				Expr("deleted_at").IsNull(),
			).And(
				Expr("id").LessThan(1),
			)).And(
				Expr("status").In([]string{"success", "failed"}),
			)),
		query: "SELECT col FROM table WHERE deleted_at IS NULL AND id < $1 AND status IN ($2, $3)",
		args:  []interface{}{1, "success", "failed"},
	}, {
		stmt:  Select("foo").Where(Expr("col").Op("op", "value")),
		query: "SELECT foo WHERE col op value",
	}, {
		stmt:  Select("foo").OrderBy("first", Call("func", Bind(1))),
		query: "SELECT foo ORDER BY first, func($1)",
		args:  []interface{}{1},
	}, {
		stmt:  Select("foo").From("table", As(Call("func", Bind(1)), "alias")),
		query: "SELECT foo FROM table, func($1) AS alias",
		args:  []interface{}{1},
	}} {
		query, args := tt.stmt.Build()
		assertf(t, query == tt.query, "expected %q, got %q", tt.query, query)
		assertf(t, len(args) == len(tt.args), "expected %d args, got %d", len(tt.args), len(args))
		minlen := len(args)
		if len(tt.args) < minlen {
			minlen = len(tt.args)
		}
		for i := 0; i < minlen; i++ {
			assertf(t, args[i] == tt.args[i], "expected %#v, got %#v", tt.args[i], args[i])
		}
	}
}
