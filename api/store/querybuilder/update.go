package querybuilder

// Update returns a new update statement.
func Update(table string) *UpdateStmt {
	return &UpdateStmt{table: table}
}

// A UpdateStmt is a update statement.
type UpdateStmt struct {
	table     string
	set       *set
	where     *where
	returning *returning
}

// Build returns the built statement and its parameters.
func (stmt *UpdateStmt) Build() (string, []interface{}) {
	b := new(builder)

	b.write("UPDATE " + stmt.table + " ")
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
func (stmt *UpdateStmt) Set(set map[string]interface{}) *UpdateStmt {
	stmt.set = newSet(set)
	return stmt
}

// Where adds a where clause.
func (stmt *UpdateStmt) Where(where interface{}) *UpdateStmt {
	stmt.where = newWhere(where)
	return stmt
}

// Returning adds a returning clause.
func (stmt *UpdateStmt) Returning(values ...string) *UpdateStmt {
	stmt.returning = newReturning(values...)
	return stmt
}

func newSet(m map[string]interface{}) *set {
	exprs := make(map[string]Expression, len(m))
	for k, v := range m {
		exprs[k] = newBindExpression(v)
	}
	return &set{exprs: exprs}
}

type set struct{ exprs map[string]Expression }

func (set set) build(b *builder) {
	b.write("SET ")
	var needcomma bool
	for k, expr := range set.exprs {
		if needcomma {
			b.write(", ")
		}
		b.write(k + " = ")
		expr.build(b)

		needcomma = true
	}
}
