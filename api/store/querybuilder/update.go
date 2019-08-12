package querybuilder

// NewUpdate returns a new update statement.
func NewUpdate(table string) *Update {
	return &Update{table: table}
}

// A Update is a update statement.
type Update struct {
	table     string
	set       *set
	where     *where
	returning *returning
}

// Build returns the built statement and its parameters.
func (stmt *Update) Build() (string, []interface{}) {
	b := new(builder)

	b.write("UPDATE ")
	b.write(stmt.table)
	b.write(" ")
	stmt.set.build(b)

	if stmt.where != nil {
		b.write(" ")
		stmt.where.build(b)
	}

	if stmt.returning != nil {
		b.write(" ")
		stmt.returning.build(b)
	}

	return b.buf.String(), b.params
}

// Set adds a set clause.
func (stmt *Update) Set(set map[string]interface{}) *Update {
	stmt.set = newSet(set)
	return stmt
}

// Where adds a where clause.
func (stmt *Update) Where(e Expr) *Update {
	stmt.where = newWhere(e)
	return stmt
}

// Returning adds a returning clause.
func (stmt *Update) Returning(values ...string) *Update {
	stmt.returning = newReturning(values...)
	return stmt
}

func newSet(m map[string]interface{}) *set {
	exprs := make(map[string]Expr, len(m))
	for k, v := range m {
		if expr, ok := v.(Expr); ok {
			exprs[k] = expr
		} else {
			exprs[k] = NewBindValue(v)
		}
	}
	return &set{exprs: exprs}
}

type set struct{ exprs map[string]Expr }

func (set set) build(b *builder) {
	b.write("SET ")
	var needcomma bool
	for k, expr := range set.exprs {
		if needcomma {
			b.write(", ")
		}
		b.write(k)
		b.write(" = ")
		expr.build(b)

		needcomma = true
	}
}
