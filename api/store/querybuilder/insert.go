package querybuilder

// NewInsert returns a new insert statement.
func NewInsert(table string, columns []string) *Insert {
	return &Insert{table: table, columns: columns}
}

// A Insert is a insert statement.
type Insert struct {
	table     string
	columns   []string
	values    *values
	returning *returning
}

// Values adds a values clause.
func (stmt *Insert) Values(values ...interface{}) *Insert {
	stmt.values = newValues(values...)
	return stmt
}

// Returning adds a returning clause.
func (stmt *Insert) Returning(values ...string) *Insert {
	stmt.returning = newReturning(values...)
	return stmt
}

// Build returns the built statement and its parameters.
func (stmt *Insert) Build() (string, []interface{}) {
	b := new(builder)
	b.write("INSERT INTO ")
	b.write(stmt.table)
	b.write("(")
	for i := range stmt.columns {
		if i > 0 {
			b.write(", ")
		}
		b.write(stmt.columns[i])
	}
	b.write(")")

	if stmt.values != nil {
		b.write(" ")
		stmt.values.build(b)
	}

	if stmt.returning != nil {
		b.write(" ")
		stmt.returning.build(b)
	}

	return b.buf.String(), b.params
}

func newValues(v ...interface{}) *values { return &values{v: v} }

type values struct{ v []interface{} }

func (values values) build(b *builder) {
	b.write("VALUES(")
	for i := range values.v {
		if i > 0 {
			b.write(", ")
		}
		b.bind(values.v[i])
	}
	b.write(")")
}

func newReturning(v ...string) *returning { return &returning{v: v} }

type returning struct{ v []string }

func (returning returning) build(b *builder) {
	b.write("RETURNING ")
	for i := range returning.v {
		if i > 0 {
			b.write(", ")
		}
		b.write(returning.v[i])
	}
	b.write("")
}