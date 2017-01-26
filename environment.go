package main

import "fmt"

type Environment struct {
	parent *Environment
	values map[string]Value
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{
		parent: parent,
		values: map[string]Value{},
	}
}

func NewRootEnvironment() *Environment {
	env := NewEnvironment(nil)
	AddBuiltins(env)
	return env
}

func (v *Environment) Eval(env *Environment) Value {
	return v
}

func (v *Environment) Type() string {
	return "environment"
}

func (v *Environment) String() string {
	return fmt.Sprintf("#<environment %p>", v)
}

func (e *Environment) Get(k string) Value {
	if v, ok := e.values[k]; ok {
		return v
	} else if e.parent != nil {
		return e.parent.Get(k)
	} else {
		return NULL
	}
}

func (e *Environment) Set(k string, v Value) {
	e.values[k] = v
}
