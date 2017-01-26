package main

import "strconv"

type Value interface {
	Eval(env Environment) Value
	String() string
}

// Sym
// =======================

type Sym struct {
	name string
}

func NewSym(name string) Value {
	return &Sym{name: name}
}

func (v *Sym) Eval(env Environment) Value {
	return v
}

func (v *Sym) String() string {
	return v.name
}

// Null
// =======================

type Null struct{}

var NULL = Null{}

func (v Null) Eval(env Environment) Value {
	return v
}

func (v Null) Type() string {
	return "null"
}

func (v Null) String() string {
	return "null"
}

// Boolean
// =======================

type Bool bool

var TRUE = Bool(true)
var FALSE = Bool(false)

func (v Bool) Eval(env Environment) Value {
	return v
}

func (v Bool) Type() string {
	return "boolean"
}

func (v Bool) String() string {
	return strconv.FormatBool(bool(v))
}

// String
// =======================

type String string

func NewString(value string) Value {
	return String(value)
}

func (v String) Eval(env Environment) Value {
	return v
}

func (v String) String() string {
	return strconv.Quote(string(v))
}

// Int
// =======================

type Int int64

func NewInt(value int64) Value {
	return Int(value)
}

func (v Int) Eval(env Environment) Value {
	return v
}

func (v Int) String() string {
	return strconv.FormatInt(int64(v), 10)
}

// Float
// =======================

type Float float64

func NewFloat(value float64) Value {
	return Float(value)
}

func (v Float) Eval(env Environment) Value {
	return v
}

func (v Float) String() string {
	return strconv.FormatFloat(float64(v), 'f', -1, 64)
}

// Cell
// =======================

type Cell struct {
	values []Value
}

func NewCell(values []Value) Value {
	return &Cell{values: values}
}

func (s *Cell) Eval(env Environment) Value {
	return s
}

func (s *Cell) String() string {
	formatted := "("
	for i, v := range s.values {
		if i > 0 {
			formatted += " "
		}
		formatted += v.String()
	}
	return formatted + ")"
}
