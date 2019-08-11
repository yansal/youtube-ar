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
	return &infixExpr{left: newIdentifier(i.identifier), op: "IN", right: newArray(array)}
}

// LessThan returns a new Expr.
func (i *Identifier) LessThan(value interface{}) Expr {
	return &infixExpr{left: newIdentifier(i.identifier), op: "<", right: newValue(value)}
}

// Equal returns a new Expr.
func (i *Identifier) Equal(value interface{}) Expr {
	return &infixExpr{left: newIdentifier(i.identifier), op: "=", right: newValue(value)}
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

type array []interface{}

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

func newValue(v interface{}) Expr {
	return value{v: v}
}

type value struct {
	v interface{}
}

func (v value) build(b *builder) {
	b.bind(v.v)
}

func newIdentifier(i string) Expr {
	return identifier{s: i}
}

type identifier struct {
	s string
}

func (i identifier) build(b *builder) {
	b.write(i.s)
}
