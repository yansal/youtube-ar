package querybuilder

// NewSelect returns a new select statement.
func NewSelect(columns ...interface{}) *Select {
	return &Select{columns: newColumns(columns...)}
}

// A Select is a select statement.
type Select struct {
	columns *columns
	from    *from
	where   *where
	orderby *orderby
	limit   *limit
	offset  *offset
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

// Offset adds an offset.
func (stmt *Select) Offset(i int64) *Select {
	stmt.offset = newOffset(i)
	return stmt
}

// Build returns the built statement and its parameters.
func (stmt *Select) Build() (string, []interface{}) {
	b := new(builder)
	b.write("SELECT ")
	stmt.columns.build(b)

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

	if stmt.offset != nil {
		b.write(" ")
		stmt.offset.build(b)
	}

	return b.buf.String(), b.params
}

func newColumns(c ...interface{}) *columns {
	exprs := make([]Expr, 0, len(c))
	for _, col := range c {
		if expr, ok := col.(Expr); ok {
			exprs = append(exprs, expr)
		} else {
			exprs = append(exprs, newValue(col))
		}
	}
	return &columns{exprs: exprs}
}

type columns struct{ exprs []Expr }

func (c columns) build(b *builder) {
	for i := range c.exprs {
		if i > 0 {
			b.write(", ")
		}
		c.exprs[i].build(b)
	}
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

func newOffset(i int64) *offset { return &offset{i: i} }

type offset struct{ i int64 }

func (offset offset) build(b *builder) {
	b.write("OFFSET")
	b.write(" ")
	b.bind(offset.i)
}
