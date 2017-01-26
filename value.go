package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

type Value interface {
	Eval(env *Environment) Value
	Type() string
	String() string
}

// Sym
// =======================

type Sym struct {
	Name string
}

func NewSym(name string) Value {
	return &Sym{Name: name}
}

func (v *Sym) Eval(env *Environment) Value {
	return v
}

func (v *Sym) Type() string {
	return "symbol"
}

func (v *Sym) String() string {
	return v.Name
}

// Null
// =======================

type Null struct{}

var NULL = Null{}

func (v Null) Eval(env *Environment) Value {
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

func (v Bool) Eval(env *Environment) Value {
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

func (v String) Eval(env *Environment) Value {
	return v
}

func (v String) Type() string {
	return "string"
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

func (v Int) Eval(env *Environment) Value {
	return v
}

func (v Int) Type() string {
	return "integer"
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

func (v Float) Eval(env *Environment) Value {
	return v
}

func (v Float) Type() string {
	return "float"
}

func (v Float) String() string {
	return strconv.FormatFloat(float64(v), 'f', -1, 64)
}

// Cell
// =======================

type Cell struct {
	Values []Value
}

func NewCell(values []Value) Value {
	return &Cell{Values: values}
}

func (v *Cell) Eval(env *Environment) Value {
	if os.Getenv("DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "debug: eval: %s\n", v)
	}
	if len(v.Values) == 0 {
		return v
	}

	head := v.Values[0]
	if head.Type() == "symbol" {
		name := head.String()
		head = env.Get(name)
		if head.Type() == "null" {
			panic(fmt.Sprintf("trying to call unbound symbol '%s' in '%s'", name, v))
		}
	}

	switch head.Type() {
	case "builtin":
		return head.(*Builtin).Apply(env, v.Values[1:])
	case "closure":
		return head.(*Closure).Apply(env, v.Values[1:])
	default:
		panic(fmt.Sprintf("trying to call non-callable value `%s'", head.String()))
	}
}

func (v Cell) Type() string {
	return "list"
}

func (s *Cell) String() string {
	formatted := "("
	for i, v := range s.Values {
		if i > 0 {
			formatted += " "
		}
		formatted += v.String()
	}
	return formatted + ")"
}

// Stream
// =======================

type Stream struct {
	Value io.ReadWriteCloser
}

func NewStream(v io.ReadWriteCloser) Value {
	return &Stream{Value: v}
}

func (v *Stream) Write(bs []byte) {
	if _, err := v.Value.Write(bs); err != nil {
		panic(fmt.Sprintf("error writing to stream: %s", err))
	}
}

func (v *Stream) Eval(env *Environment) Value {
	return v
}

func (v *Stream) Type() string {
	return "stream"
}

func (v *Stream) String() string {
	return fmt.Sprintf("#<stream %p>", v)
}

// Builtin
// =======================

type BuiltinFn func(*Environment, []Value) Value
type Builtin struct {
	Name      string
	Fn        BuiltinFn
	IsSpecial bool
}

func NewBuiltin(name string, fn BuiltinFn) Value {
	return &Builtin{Name: name, Fn: fn, IsSpecial: false}
}

func (v *Builtin) Apply(env *Environment, args []Value) Value {
	if v.IsSpecial {
		return (v.Fn)(env, args)
	} else {
		evaluatedArgs := []Value{}
		for _, arg := range args {
			evaluatedArgs = append(evaluatedArgs, FullEval(env, arg))
		}
		return (v.Fn)(env, evaluatedArgs)
	}
}

func (v *Builtin) Eval(env *Environment) Value {
	return v
}

func (v *Builtin) Type() string {
	return "builtin"
}

func (v *Builtin) String() string {
	return fmt.Sprintf("#<builtin %s>", v.Name)
}

// Closure
// =======================

type Closure struct {
	Env      *Environment
	ArgNames []string
	Body     []Value
}

func NewClosure(env *Environment, argNames []string, body []Value) Value {
	return &Closure{Env: env, ArgNames: argNames, Body: body}
}

func (v *Closure) Apply(env *Environment, args []Value) Value {
	evalEnv := NewEnvironment(v.Env)

	rest := ""
	restValues := []Value{}
	argNames := v.ArgNames

	for _, arg := range args {
		if len(argNames) == 0 && rest == "" {
			panic(fmt.Sprintf(
				"function called with too many arguments: wanted %v, got %v. Args: %v",
				len(v.ArgNames), len(args), NewCell(args),
			))
		} else if rest != "" {
			restValues = append(restValues, FullEval(env, arg))
		} else if argNames[0] == "&" {
			if len(argNames) != 2 {
				panic(fmt.Sprintf("found illegal '&' in argument list: %s", NewCell(args)))
			}
			rest = argNames[1]
			argNames = []string{}
			restValues = append(restValues, FullEval(env, arg))
		} else {
			evalEnv.Set(argNames[0], FullEval(env, arg))
			argNames = argNames[1:]
		}
	}

	if rest != "" {
		evalEnv.Set(rest, NewCell(restValues))
	}

	if len(argNames) == 0 {
		// Are we done filling required args? Then apply function
		return NewCell(append([]Value{NewSym("do")}, v.Body...)).Eval(evalEnv)
	} else {
		// Create a new closure for the partially applied function
		return NewClosure(evalEnv, argNames, v.Body)
	}
}

func (v *Closure) Eval(env *Environment) Value {
	return v
}

func (v *Closure) Type() string {
	return "closure"
}

func (v *Closure) String() string {
	return fmt.Sprintf("#<closure %p>", v)
}
