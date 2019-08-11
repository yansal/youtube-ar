package querybuilder

// NewSelect returns a new select statement.
func NewSelect(columns ...string) *Select {
	return &Select{columns: columns}
}

// A Select is a select statement.
type Select struct {
	columns []string
	from    *from
	where   *where
	orderby *orderby
	limit   *limit
}

// From adds a from clause.
func (stmt *Select) From(table string) *Select {
	stmt.from = newFrom(table)
	return stmt
}

// Where adds a where clause.
func (stmt *Select) Where(e Expr) *Select {
	stmt.where = newWhere(e)
	return stmt
}

// OrderBy adds an order by.
func (stmt *Select) OrderBy(s string) *Select {
	stmt.orderby = newOrderBy(s)
	return stmt
}

// Limit adds a limit.
func (stmt *Select) Limit(i int64) *Select {
	stmt.limit = newLimit(i)
	return stmt
}

// Build returns the built statement and its parameters.
func (stmt *Select) Build() (string, []interface{}) {
	if len(stmt.columns) == 0 {
		panic("no columns")
	}

	b := new(builder)
	b.write("SELECT ")
	for i := range stmt.columns {
		if i > 0 {
			b.write(", ")
		}
		b.write(stmt.columns[i])
	}

	if stmt.from != nil {
		b.write(" ")
		stmt.from.build(b)
	}

	if stmt.where != nil {
		b.write(" ")
		stmt.where.build(b)
	}

	if stmt.orderby != nil {
		b.write(" ")
		stmt.orderby.build(b)
	}

	if stmt.limit != nil {
		b.write(" ")
		stmt.limit.build(b)
	}

	return b.buf.String(), b.params
}

func newFrom(table string) *from { return &from{table: table} }

type from struct{ table string }

func (from from) build(b *builder) {
	b.write("FROM")
	b.write(" ")
	b.write(from.table)
}

func newWhere(expr Expr) *where { return &where{expr: expr} }

type where struct{ expr Expr }

func (where where) build(b *builder) {
	b.write("WHERE")
	b.write(" ")
	where.expr.build(b)
}

func newOrderBy(s string) *orderby { return &orderby{s: s} }

type orderby struct{ s string }

func (orderby orderby) build(b *builder) {
	b.write("ORDER BY")
	b.write(" ")
	b.write(orderby.s)
}

func newLimit(i int64) *limit { return &limit{i: i} }

type limit struct{ i int64 }

func (limit limit) build(b *builder) {
	b.write("LIMIT")
	b.write(" ")
	b.bind(limit.i)
}
