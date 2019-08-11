package querybuilder

import "fmt"

// Expr is an expression.
type Expr interface {
	build(*builder)
}

// NewIdentifier returns a new identifier.
func NewIdentifier(identifier string) *Identifier {
	return &Identifier{identifier: identifier}
}

// Identifier is an identifier.
type Identifier struct {
	identifier string
}

// In returns a new Expr.
func (i *Identifier) In(array interface{}) Expr {
	return &infixExpr{left: newValue(i.identifier), op: "IN", right: newArray(array)}
}

// LessThan returns a new Expr.
func (i *Identifier) LessThan(value interface{}) Expr {
	return &infixExpr{left: newValue(i.identifier), op: "<", right: NewBindValue(value)}
}

// Equal returns a new Expr.
func (i *Identifier) Equal(value interface{}) Expr {
	return &infixExpr{left: newValue(i.identifier), op: "=", right: NewBindValue(value)}
}

// NewBoolExpr returns a new boolean expression.
func NewBoolExpr(left Expr) *BoolExpr {
	return &BoolExpr{left: left}
}

// BoolExpr is a boolean expression.
type BoolExpr struct {
	left Expr
}

// And returns a new Expr.
func (i *BoolExpr) And(right Expr) Expr {
	return &infixExpr{left: i.left, op: "AND", right: right}
}

// NewCallExpr returns a new call expression.
func NewCallExpr(fun string, args ...interface{}) Expr {
	callargs := make([]Expr, 0, len(args))
	for _, arg := range args {
		if expr, ok := arg.(Expr); ok {
			callargs = append(callargs, expr)
		} else {
			callargs = append(callargs, newValue(arg))
		}
	}
	return &callExpr{fun: fun, args: callargs}
}

type callExpr struct {
	fun  string
	args []Expr
}

func (e callExpr) build(b *builder) {
	b.write(e.fun)
	b.write("(")
	for i, arg := range e.args {
		if i > 0 {
			b.write(", ")
		}
		arg.build(b)
	}
	b.write(")")
}

// NewIndexExpr returns a new index expression.
func NewIndexExpr(array string, lower interface{}) Expr {
	var lowerexpr Expr
	if expr, ok := lower.(Expr); ok {
		lowerexpr = expr
	} else {
		lowerexpr = newValue(lower)
	}
	return &indexExpr{array: array, lower: lowerexpr}
}

type indexExpr struct {
	array string
	lower Expr
}

func (e indexExpr) build(b *builder) {
	b.write(e.array)
	b.write("[")
	e.lower.build(b)
	b.write(":]")
}

func newArray(value interface{}) array {
	var array array
	switch v := value.(type) {
	case []string:
		for i := range v {
			array = append(array, v[i])
		}
	default:
		panic(fmt.Sprintf("don't know how to create an array from a value of type %T", value))
	}
	return array
}

type array []string

func (a array) build(b *builder) {
	b.write("(")
	for i := range a {
		if i > 0 {
			b.write(", ")
		}
		b.bind(a[i])
	}
	b.write(")")
}

type infixExpr struct {
	left  Expr
	op    string
	right Expr
}

func (e infixExpr) build(b *builder) {
	e.left.build(b)
	b.write(" ")
	b.write(e.op)
	b.write(" ")
	e.right.build(b)
}

// NewBindValue returns a new bind value.
func NewBindValue(v interface{}) Expr {
	return bindValue{v: v}
}

type bindValue struct {
	v interface{}
}

func (v bindValue) build(b *builder) {
	b.bind(v.v)
}

func newValue(v interface{}) Expr {
	switch vv := v.(type) {
	case string:
		return value{s: vv}
	default:
		panic(fmt.Sprintf("don't know how to write a value of type %T", v))
	}
}

type value struct {
	s string
}

func (v value) build(b *builder) {
	b.write(v.s)
}
