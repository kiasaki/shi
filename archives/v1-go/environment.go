package shi

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

func (v *Environment) Type() string {
	return "environment"
}

func (v *Environment) String() string {
	return fmt.Sprintf("#<environment %p>", v)
}

func (e *Environment) Root() *Environment {
	node := e
	for i := 0; i < 10000; i++ { // make sure we are never stuck here and panic
		if node.parent == nil {
			return node
		}
		node = node.parent
	}
	panic("env: root: unreachable")
}

func (e *Environment) Get(k string) Value {
	if k == "*env*" {
		return e
	}

	if v, ok := e.values[k]; ok {
		return v
	} else if e.parent != nil {
		return e.parent.Get(k)
	} else {
		return nil
	}
}

func (e *Environment) Set(k string, v Value) {
	e.values[k] = v
}

func (e *Environment) Symbols() []Value {
	syms := []Value{}
	for k, _ := range e.values {
		syms = append(syms, NewSym(k))
	}
	return syms
}
