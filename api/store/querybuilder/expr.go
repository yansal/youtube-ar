package querybuilder

import "fmt"

// Expression is an expression.
type Expression interface {
	build(*builder)
}

// Expr returns a new infix expression.
func Expr(left interface{}) *Infix {
	return &Infix{left: newExpression(left)}
}

func newExpression(i interface{}) Expression {
	if expr, ok := i.(Expression); ok {
		return expr
	}
	return newValue(i)
}

func newValue(v interface{}) Expression {
	switch vv := v.(type) {
	case string:
		return value{s: vv}
	default:
		panic(fmt.Sprintf("don't know how to write value %#v", v))
	}
}

type value struct {
	s string
}

func (v value) build(b *builder) {
	b.write(v.s)
}

func newBindExpression(i interface{}) Expression {
	if expr, ok := i.(Expression); ok {
		return expr
	}
	return Bind(i)
}

// Bind returns a new bind value.
func Bind(v interface{}) Expression {
	return bindValue{v: v}
}

type bindValue struct {
	v interface{}
}

func (v bindValue) build(b *builder) {
	b.bind(v.v)
}

// Infix is an infix expression.
type Infix struct {
	left Expression
}

// In returns a new Expression.
func (i *Infix) In(array interface{}) Expression {
	return &infixExpr{left: i.left, op: "IN", right: newArray(array)}
}

// LessThan returns a new Expression.
func (i *Infix) LessThan(value interface{}) Expression {
	return &infixExpr{left: i.left, op: "<", right: Bind(value)}
}

// Equal returns a new Expression.
func (i *Infix) Equal(value interface{}) Expression {
	return &infixExpr{left: i.left, op: "=", right: Bind(value)}
}

// IsNull returns a new Expression.
func (i *Infix) IsNull() Expression {
	return &infixExpr{left: i.left, op: "IS", right: newValue("NULL")}
}

// And returns a new Expression.
func (i *Infix) And(right Expression) Expression {
	return &infixExpr{left: i.left, op: "AND", right: right}
}

// Op returns a new Expression with a custom operator.
func (i *Infix) Op(op string, right interface{}) Expression {
	return &infixExpr{left: i.left, op: op, right: newExpression(right)}
}

type infixExpr struct {
	left  Expression
	op    string
	right Expression
}

func (e infixExpr) build(b *builder) {
	e.left.build(b)
	b.write(" ")
	b.write(e.op)
	b.write(" ")
	e.right.build(b)
}

// Call returns a new call expression.
func Call(fun string, args ...interface{}) Expression {
	callargs := make([]Expression, 0, len(args))
	for _, arg := range args {
		if expr, ok := arg.(Expression); ok {
			callargs = append(callargs, expr)
		} else {
			callargs = append(callargs, newValue(arg))
		}
	}
	return &callExpr{fun: fun, args: callargs}
}

type callExpr struct {
	fun  string
	args []Expression
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

// Index returns a new index expression.
func Index(array string, lower interface{}) Expression {
	var lowerexpr Expression
	if expr, ok := lower.(Expression); ok {
		lowerexpr = expr
	} else {
		lowerexpr = newValue(lower)
	}
	return &indexExpr{array: array, lower: lowerexpr}
}

type indexExpr struct {
	array string
	lower Expression
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
