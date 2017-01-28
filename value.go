package shi

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

type Value interface {
	Type() string
	String() string
}

type ReadablyStringer interface {
	ReadableString() string
}

type Callable interface {
	Call(*Environment, []Value) Value
}

// Sym
// =======================

type Sym struct {
	Name string
}

func NewSym(name string) Value {
	return &Sym{Name: name}
}

func (*Sym) Type() string {
	return "symbol"
}

func (v *Sym) String() string {
	return v.Name
}

// Null
// =======================

type Null struct{}

var NULL = Null{}

func (Null) Type() string {
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

func (Bool) Type() string {
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

func (String) Type() string {
	return "string"
}

func (v String) String() string {
	return strconv.Quote(string(v))
}

func (v String) ReadableString() string {
	return string(v)
}

// Int
// =======================

type Int int64

func NewInt(value int64) Value {
	return Int(value)
}

func (Int) Type() string {
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

func (Float) Type() string {
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

func (*Cell) Type() string {
	return "list"
}

func (v *Cell) String() string {
	return printValues("(", ")", false, v.Values)
}

func (v *Cell) ReadableString() string {
	return printValues("(", ")", true, v.Values)
}

// Vector
// =======================

type Vector struct {
	Values []Value
}

func NewVector(values []Value) Value {
	return &Vector{Values: values}
}

func (*Vector) Type() string {
	return "vector"
}

func (v *Vector) String() string {
	return printValues("[", "]", false, v.Values)
}

func (v *Vector) ReadableString() string {
	return printValues("[", "]", true, v.Values)
}

// Stream
// =======================

type StreamDir string

const (
	StreamDirIn  StreamDir = "in"
	StreamDirOut           = "out"
)

type Stream struct {
	Direction StreamDir
	Value     io.ReadWriteCloser
	In        *bufio.Reader
	Out       *bufio.Writer
}

func NewStream(dir StreamDir, v io.ReadWriteCloser) Value {
	return &Stream{
		Direction: dir,
		Value:     v,
		In:        bufio.NewReader(v),
		Out:       bufio.NewWriter(v),
	}
}

func (*Stream) Type() string {
	return "stream"
}

func (v *Stream) String() string {
	return fmt.Sprintf("#<stream %s %p>", v.Direction, v)
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

func (v *Builtin) Call(env *Environment, args []Value) Value {
	if v.IsSpecial {
		return (v.Fn)(env, args)
	} else {
		evaluatedArgs := []Value{}
		for _, arg := range args {
			evaluatedArgs = append(evaluatedArgs, Eval(env, arg))
		}
		return (v.Fn)(env, evaluatedArgs)
	}
}

func (*Builtin) Type() string {
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

func (v *Closure) Call(env *Environment, args []Value) Value {
	argsEnv := NewEnvironment(v.Env)
	argNamesLeft := buildCallEnv(true, env, argsEnv, v.ArgNames, args)

	if len(argNamesLeft) == 0 {
		// Are we done filling required args? Then apply function
		return Eval(argsEnv, NewCell(append([]Value{NewSym("do")}, v.Body...)))
	} else {
		// Create a new closure for the partially applied function
		return NewClosure(argsEnv, argNamesLeft, v.Body)
	}
}

func (*Closure) Type() string {
	return "closure"
}

func (v *Closure) String() string {
	return fmt.Sprintf("#<closure %p>", v)
}

// Macro
// =======================

type Macro struct {
	ArgNames []string
	Body     []Value
}

func NewMacro(argNames []string, body []Value) Value {
	return &Macro{ArgNames: argNames, Body: body}
}

func (v *Macro) Call(env *Environment, args []Value) Value {
	evalEnv := NewEnvironment(env)
	argNamesLeft := buildCallEnv(false, env, evalEnv, v.ArgNames, args)

	if len(argNamesLeft) > 0 {
		panic(fmt.Sprintf("macro called without all arguments, wanted '%s', got '%s'", v.ArgNames, args))
	}

	var expandedValue Value = NULL
	for _, expr := range v.Body {
		expandedValue = Eval(evalEnv, expr)
	}
	return Eval(env, expandedValue)
}

func (*Macro) Type() string {
	return "macro"
}

func (v *Macro) String() string {
	return fmt.Sprintf("#<macro %p>", v)
}
