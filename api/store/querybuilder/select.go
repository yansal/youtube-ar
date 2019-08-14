package querybuilder

// Select returns a new select statement.
func Select(columns ...interface{}) *SelectStmt {
	return &SelectStmt{columns: newColumns(columns...)}
}

// A SelectStmt is a select statement.
type SelectStmt struct {
	columns *columns
	from    *from
	where   *where
	orderby *orderby
	limit   *limit
	offset  *offset
}

// From adds a from clause.
func (stmt *SelectStmt) From(table string) *SelectStmt {
	stmt.from = newFrom(table)
	return stmt
}

// Where adds a where clause.
func (stmt *SelectStmt) Where(where interface{}) *SelectStmt {
	stmt.where = newWhere(where)
	return stmt
}

// OrderBy adds an order by.
func (stmt *SelectStmt) OrderBy(s string) *SelectStmt {
	stmt.orderby = newOrderBy(s)
	return stmt
}

// Limit adds a limit.
func (stmt *SelectStmt) Limit(i int64) *SelectStmt {
	stmt.limit = newLimit(i)
	return stmt
}

// Offset adds an offset.
func (stmt *SelectStmt) Offset(i int64) *SelectStmt {
	stmt.offset = newOffset(i)
	return stmt
}

// Build returns the built statement and its parameters.
func (stmt *SelectStmt) Build() (string, []interface{}) {
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
	exprs := make([]Expression, 0, len(c))
	for _, col := range c {
		exprs = append(exprs, newExpression((col)))
	}
	return &columns{exprs: exprs}
}

type columns struct{ exprs []Expression }

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
	b.write("FROM " + from.table)
}

func newWhere(i interface{}) *where { return &where{expr: newExpression(i)} }

type where struct{ expr Expression }

func (where where) build(b *builder) {
	b.write("WHERE ")
	where.expr.build(b)
}

func newOrderBy(s string) *orderby { return &orderby{s: s} }

type orderby struct{ s string }

func (orderby orderby) build(b *builder) {
	b.write("ORDER BY " + orderby.s)
}

func newLimit(i int64) *limit { return &limit{i: i} }

type limit struct{ i int64 }

func (limit limit) build(b *builder) {
	b.write("LIMIT ")
	b.bind(limit.i)
}

func newOffset(i int64) *offset { return &offset{i: i} }

type offset struct{ i int64 }

func (offset offset) build(b *builder) {
	b.write("OFFSET ")
	b.bind(offset.i)
}

// Column returns a new SelectColumn.
func Column(i interface{}) *SelectColumn {
	return &SelectColumn{expr: newExpression(i)}
}

// SelectColumn is a select column.
type SelectColumn struct {
	expr Expression
}

// As adds a column alias.
func (s *SelectColumn) As(as string) Expression {
	return &column{expr: s.expr, as: as}
}

type column struct {
	expr Expression
	as   string
}

func (c *column) build(b *builder) {
	c.expr.build(b)
	b.write(" AS " + c.as)
}
