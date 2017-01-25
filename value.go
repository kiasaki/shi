package main

import (
	"fmt"
	"strconv"
)

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

func (s *Sym) Eval(env Environment) Value {
	return s
}

func (s *Sym) String() string {
	return s.name
}

// Int
// =======================

type Int struct {
	value int64
}

func NewInt(value int64) Value {
	return &Int{value: value}
}

func (s *Int) Eval(env Environment) Value {
	return s
}

func (s *Int) String() string {
	return strconv.FormatInt(s.value, 10)
}

// Float
// =======================

type Float struct {
	value float64
}

func NewFloat(value float64) Value {
	return &Float{value: value}
}

func (s *Float) Eval(env Environment) Value {
	return s
}

func (s *Float) String() string {
	return strconv.FormatFloat(s.value, 'f', -1, 64)
}

// String
// =======================

type String struct {
	value string
}

func NewString(value string) Value {
	return &String{value: value}
}

func (s *String) Eval(env Environment) Value {
	return s
}

func (s *String) String() string {
	return fmt.Sprintf("\"%s\"", strconv.Quote(s.value))
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
