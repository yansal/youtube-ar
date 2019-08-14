package querybuilder

import "testing"

func TestUpdate(t *testing.T) {
	for _, tt := range []struct {
		stmt  *UpdateStmt
		query string
		args  []interface{}
	}{{
		stmt:  Update("urls").Set(map[string]interface{}{"logs": Call("array_append", "logs", Bind(1))}),
		query: "UPDATE urls SET logs = array_append(logs, $1)",
		args:  []interface{}{1},
	}, {
		stmt:  Update("table").Set(map[string]interface{}{"foo": "bar"}).Where("bla"),
		query: "UPDATE table SET foo = $1 WHERE bla",
		args:  []interface{}{"bar"},
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
