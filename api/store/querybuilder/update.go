package querybuilder

// NewUpdate returns a new update statement.
func NewUpdate(table string, set map[string]interface{}) *Update {
	return &Update{table: table, set: newSet(set)}
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
	if stmt.set != nil {
		b.write(" ")
		stmt.set.build(b)
	}

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

func newSet(v map[string]interface{}) *set { return &set{v: v} }

type set struct{ v map[string]interface{} }

func (set set) build(b *builder) {
	b.write("SET ")
	var needcomma bool
	for k, v := range set.v {
		if needcomma {
			b.write(", ")
		}
		b.write(k)
		b.write(" = ")
		b.bind(v)
		needcomma = true
	}
}
