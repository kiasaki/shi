package shi

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var registeredModules = []func(*Environment){}

func RegisterNativeModule(fn func(*Environment)) {
	registeredModules = append(registeredModules, fn)
}

func AddBuiltin(env *Environment, name string, fn func(*Environment, []Value) Value) {
	env.Set(name, NewBuiltin(name, BuiltinFn(fn)))
}

func AddBuiltinSpecial(env *Environment, name string, fn func(*Environment, []Value) Value) {
	b := NewBuiltin(name, BuiltinFn(fn))
	b.(*Builtin).IsSpecial = true
	env.Set(name, b)
}

func AddBuiltins(env *Environment) {
	// Language
	AddBuiltinSpecial(env, "fn", builtinFn)
	AddBuiltin(env, "do", builtinDo)
	AddBuiltinSpecial(env, "cond", builtinCond)
	AddBuiltinSpecial(env, "loop", builtinLoop)
	AddBuiltin(env, "recur", builtinRecur)
	AddBuiltinSpecial(env, "quote", builtinQuote)
	AddBuiltinSpecial(env, "macro", builtinMacro)

	// Basics
	env.Set("*print-readably*", TRUE)
	AddBuiltin(env, "pr-str", builtinPrStr)
	AddBuiltin(env, "type", builtinType)
	AddBuiltin(env, "parse", builtinParse)
	AddBuiltin(env, "eval", builtinEval)
	AddBuiltin(env, "load", builtinLoad)

	// Errors
	AddBuiltin(env, "error", builtinError)
	AddBuiltin(env, "trap-error", builtinTrapError)

	// Environments
	AddBuiltin(env, "environment", builtinEnvironment)
	AddBuiltin(env, "root-environment", builtinRootEnvironment)
	AddBuiltin(env, "environment-set", builtinEnvironmentSet)
	AddBuiltin(env, "environment-get", builtinEnvironmentGet)
	AddBuiltin(env, "environment-root", builtinEnvironmentRoot)
	AddBuiltin(env, "environment-symbols", builtinEnvironmentSymbols)

	// Compare
	AddBuiltin(env, "eq", builtinEq)
	AddBuiltin(env, "eql", builtinEql)

	// Symbols
	AddBuiltin(env, "sym", builtinSym)

	// Strings
	AddBuiltin(env, "str", builtinStr)
	AddBuiltin(env, "str-join", builtinStrJoin)
	AddBuiltin(env, "str-split", builtinStrSplit)

	// Lists
	AddBuiltin(env, "list", builtinList)
	AddBuiltin(env, "list-nth", builtinListNth)
	AddBuiltin(env, "list-join", builtinListJoin)
	AddBuiltin(env, "list-slice", builtinListSlice)
	AddBuiltin(env, "list-length", builtinListLength)

	// Vectors
	AddBuiltin(env, "vec", builtinVec)
	AddBuiltin(env, "vec->list", builtinVecToList)

	// Math
	AddBuiltin(env, "<", builtinSmaller)
	AddBuiltin(env, "+", builtinPlus)
	AddBuiltin(env, "-", builtinMinus)
	AddBuiltin(env, "/", builtinQuotient)
	AddBuiltin(env, "*", builtinProduct)
	AddBuiltin(env, "mod", builtinMod)

	// Streams / IO
	AddBuiltin(env, "read", builtinRead)
	AddBuiltin(env, "write", builtinWrite)
	env.Set("*stdin*", NewStream(StreamDirIn, os.Stdin))
	env.Set("*stdout*", NewStream(StreamDirOut, os.Stdout))
	env.Set("*stderr*", NewStream(StreamDirOut, os.Stderr))

	// OS
	AddBuiltin(env, "exit", builtinExit)

	for _, registeredModule := range registeredModules {
		registeredModule(env)
	}
}

// Language
// =======================

func builtinFn(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, -1)

	name := ""
	if vals[0].Type() == "symbol" {
		name = vals[0].(*Sym).Name
		AssetArgsSize(vals, 2, -1)
		vals = vals[1:]
	}

	AssetArgType(vals[0], "list")
	AssetArgListType(vals[0], "symbol")

	argNames := []string{}
	for _, arg := range vals[0].(*Cell).Values {
		argNames = append(argNames, arg.String())
	}
	return NewClosure(env, name, argNames, vals[1:])
}

func builtinDo(env *Environment, vals []Value) Value {
	if len(vals) == 0 {
		return NULL
	} else {
		return vals[len(vals)-1]
	}
}

func builtinCond(env *Environment, vals []Value) Value {
	pairs := groupValsAsPairs("cond", vals)
	for _, pair := range pairs {
		// Special 'else' case
		if pair[0].Type() == "symbol" && pair[0].String() == "else" {
			return Eval(env, pair[1])
		}

		value := Eval(env, pair[0])
		if value != FALSE && value != NULL {
			// Condition test passed, return branch value
			return Eval(env, pair[1])
		}
	}
	return NULL
}

func builtinLoop(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, -1)
	AssetArgType(vals[0], "list")

	letNames := []string{}
	letValues := []Value{}
	for i, letVal := range vals[0].(*Cell).Values {
		if i%2 == 0 {
			if letVal.Type() != "symbol" {
				panic(fmt.Sprintf("loop: given a non-symbol value name: %s", letVal))
			}
			letNames = append(letNames, letVal.(*Sym).Name)
		} else {
			letValues = append(letValues, letVal)
		}
	}

	argsEnv := NewEnvironment(env)
	buildCallEnv(true, env, argsEnv, letNames, letValues)

	// Tail Call Optimized loop
	result := Value(NULL)
	for {
		// Execute body updating result
		for _, expr := range vals[1:] {
			result = Eval(argsEnv, expr)
		}
		// If result is a recur call, loop again, else return
		if result.Type() != "list" || result.(*Cell).Values[0].String() != "recur" {
			return result
		}

		// Update environment with new values from recur call
		argsLeft := buildCallEnv(false, argsEnv, argsEnv, letNames, result.(*Cell).Values[1:])
		if len(argsLeft) > 0 {
			panic(fmt.Sprintf(
				"recur: called with %d values, loop declared %d",
				len(result.(*Cell).Values)-1, len(letNames),
			))
		}
	}
}

func builtinRecur(env *Environment, vals []Value) Value {
	return BuildCall("recur", vals)
}

func builtinQuote(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)

	return vals[0]
}

func builtinMacro(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, -1)

	name := ""
	if vals[0].Type() == "symbol" {
		name = vals[0].(*Sym).Name
		AssetArgsSize(vals, 2, -1)
		vals = vals[1:]
	}

	AssetArgType(vals[0], "list")
	AssetArgListType(vals[0], "symbol")

	argNames := []string{}
	for _, arg := range vals[0].(*Cell).Values {
		argNames = append(argNames, arg.String())
	}

	return NewMacro(name, argNames, vals[1:])
}

// Basics
// =======================

func builtinPrStr(env *Environment, vals []Value) Value {
	printReadablyVal := env.Get("*print-readably*")
	printReadably := printReadablyVal != NULL && printReadablyVal != FALSE
	return NewString(printValues("", "", printReadably, vals))
}

func builtinType(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)

	return NewString(vals[0].Type())
}

func builtinParse(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 2)

	name := NewString("unknown")
	contents := vals[0]
	if len(vals) == 2 {
		name = vals[0]
		contents = vals[1]
	}
	AssetArgType(name, "string")
	AssetArgType(contents, "string")

	topLevel := Parse(string(name.(String)), string(contents.(String)))

	if len(topLevel) == 1 {
		return topLevel[0]
	} else {
		// Wrap topLevel in a do so it can be passed as a value
		return NewCell(append([]Value{NewSym("do")}, topLevel...))
	}
}

func builtinEval(env *Environment, vals []Value) Value {
	if len(vals) == 1 {
		return Eval(env, vals[0])
	}

	AssetArgsSize(vals, 2, 2)

	e := Eval(env, vals[0])
	AssetArgType(e, "environment")

	return Eval(e.(*Environment), vals[1])
}

func builtinLoadHelper(env *Environment, file string) bool {
	targetFile := file
	if len(targetFile) < 4 || targetFile[len(targetFile)-4:] != ".shi" {
		// Ensure extension is present
		targetFile = targetFile + ".shi"
	}

	targetModule := filepath.Join(file, filepath.Base(targetFile))

	if _, err := os.Stat(targetFile); err == nil {
		// Try file (i.e. ./util.shi or shi/core.shi)
		Eval(env, ParseFile(targetFile))
		return true
	} else if _, err := os.Stat(targetModule); err == nil {
		// Try module folder (i.e. shi/core/core.shi)
		Eval(env, ParseFile(targetModule))
		return true
	}
	return false
}

func builtinLoad(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)
	targetFileVal := vals[0]
	if targetFileVal.Type() == "symbol" {
		targetFileVal = NewString(targetFileVal.String())
	}
	AssetArgType(targetFileVal, "string")

	targetFile := string(targetFileVal.(String))

	if len(targetFile) == 0 {
		// No file could match, error
		panic("load: emply module or file name given")
	}
	if targetFile[0] == '.' {
		// Relative file
		if builtinLoadHelper(env, targetFile) {
			return NULL
		} else {
			panic(fmt.Sprintf("load: could not find file '%s'", targetFile))
		}
	}

	// Look for module in *shi-path* folders
	shiPathsVal := Eval(env, env.Get("*shi-path*"))
	if shiPathsVal.Type() == "cell" {
		panic("load: expected '*shi-path*' to be a string list")
	}
	shiPaths := shiPathsVal.(*Cell).Values
	targetModule := strings.Replace(targetFile, ":", string(os.PathSeparator), -1)

	for _, p := range shiPaths {
		if builtinLoadHelper(env, filepath.Join(string(p.(String)), targetModule)) {
			return NULL
		}
	}

	panic(fmt.Sprintf("load: could not find module '%s'", targetFile))
}

func BuiltinLoad(env *Environment, vals []Value) Value {
	return builtinLoad(env, vals)
}

// Error
// =======================

func builtinError(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)
	AssetArgType(vals[0], "string")
	panic(string(vals[0].(String)))
}

func builtinTrapError(env *Environment, vals []Value) (ret Value) {
	AssetArgsSize(vals, 2, 2)
	AssetArgType(vals[0], "closure")
	AssetArgType(vals[1], "closure")

	defer func() {
		if r := recover(); r != nil {
			ret = Eval(env, NewCell([]Value{vals[1], NewString(fmt.Sprintf("%v", r))}))
		}
	}()

	ret = Eval(env, NewCell([]Value{vals[0]}))
	return
}

// Environment
// =======================

func builtinEnvironment(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)
	AssetArgType(vals[0], "environment")

	return NewEnvironment(vals[0].(*Environment))
}

func builtinRootEnvironment(env *Environment, vals []Value) Value {
	return NewRootEnvironment()
}

func builtinEnvironmentSet(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 3)

	var e = env
	var k = vals[0]
	var v = vals[1]

	if len(vals) == 3 {
		givenEnv := vals[0]
		AssetArgType(givenEnv, "environment")
		e = givenEnv.(*Environment)
		k = vals[1]
		v = vals[2]
	}
	AssetArgType(k, "symbol|string")

	switch kvalue := k.(type) {
	case String:
		e.Set(string(kvalue), v)
	case *Sym:
		e.Set(kvalue.Name, v)
	}
	return v
}

func builtinEnvironmentGet(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 2)

	var e = env
	var k Value

	if len(vals) > 0 && vals[0].Type() == "environment" {
		e = vals[0].(*Environment)
		AssetArgsSize(vals, 2, 2)
		AssetArgType(vals[1], "symbol")
		k = vals[1]
	} else {
		AssetArgsSize(vals, 1, 1)
		AssetArgType(vals[0], "symbol")
		k = vals[0]
	}

	ret := e.Get(k.String())
	if ret == nil {
		return NULL
	}
	return ret
}

func builtinEnvironmentRoot(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 0, 1)
	e := env
	if len(vals) > 0 && vals[0].Type() == "environment" {
		e = vals[0].(*Environment)
	}

	return e.Root()
}

func builtinEnvironmentSymbols(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 0, 1)
	e := env
	if len(vals) > 0 && vals[0].Type() == "environment" {
		e = vals[0].(*Environment)
	}

	return NewCell(e.Symbols())
}

// Compare
// =======================

// = by value
func builtinEq(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)

	if vals[0].Type() == "symbol" && vals[1].Type() == "symbol" {
		return NewBool(vals[0].(*Sym).Name == vals[1].(*Sym).Name)
	}

	// TODO use Comparable interface
	if a, ok := vals[0].(Comparable); ok {
		return NewBool(a.Compare(vals[1]) == 0)
	} else {
		return NewBool(vals[0] == vals[1])
	}
}

// = by pointer
func builtinEql(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)

	return NewBool(vals[0].String() == vals[1].String())
}

// Symbols
// =======================

func builtinSym(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)
	AssetArgType(vals[0], "string")

	return NewSym(string(vals[0].(String)))
}

// Strings
// =======================

func builtinStr(env *Environment, vals []Value) Value {
	for i, v := range vals {
		switch vv := v.(type) {
		case String:
			vals[i] = NewString(string(vv))
		case *Sym:
			vals[i] = NewString(vv.Name)
		case Int:
			vals[i] = NewString(vv.String())
		case Float:
			vals[i] = NewString(vv.String())
		default:
			panic("str: can't use type '" + v.Type() + "' argument to str")
		}
	}
	return builtinStrJoin(env, []Value{NewString(""), NewCell(vals)})
}

func builtinStrJoin(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)
	AssetArgType(vals[0], "string")
	AssetArgType(vals[1], "list")
	AssetArgListType(vals[1], "string")

	str := ""
	for i, v := range vals[1].(*Cell).Values {
		if i > 0 {
			str += string(vals[0].(String))
		}
		str += string(v.(String))
	}
	return NewString(str)
}

func builtinStrSplit(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)
	AssetArgType(vals[0], "string")
	AssetArgType(vals[1], "string")

	sep := string(vals[0].(String))
	str := string(vals[1].(String))

	parts := []Value{}
	for _, v := range strings.Split(str, sep) {
		parts = append(parts, NewString(v))
	}
	return NewCell(parts)
}

// Lists
// =======================

func builtinList(env *Environment, vals []Value) Value {
	return NewCell(vals)
}

func builtinListNth(env *Environment, vals []Value) Value {
	AssetArgType(vals[0], "integer")
	AssetArgType(vals[1], "list")

	index := int(vals[0].(Int))
	list := vals[1].(*Cell).Values

	return list[index]
}

func builtinListJoin(env *Environment, vals []Value) Value {
	AssetArgListType(NewCell(vals), "list")

	joinedVals := []Value{}
	for _, v := range vals {
		joinedVals = append(joinedVals, v.(*Cell).Values...)
	}

	return NewCell(joinedVals)
}

func builtinListSlice(env *Environment, vals []Value) Value {
	if len(vals) == 2 {
		AssetArgType(vals[0], "integer")
		AssetArgType(vals[1], "list")

		start := int(vals[0].(Int))
		list := vals[1].(*Cell).Values

		if start >= len(list) {
			return NewCell([]Value{})
		}
		return NewCell(list[start:])
	}

	AssetArgsSize(vals, 3, 3)
	AssetArgType(vals[0], "integer")
	AssetArgType(vals[1], "integer")
	AssetArgType(vals[2], "list")

	start := int(vals[0].(Int))
	end := int(vals[1].(Int))
	list := vals[2].(*Cell).Values

	return NewCell(list[start:end])
}

func builtinListLength(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)
	AssetArgType(vals[0], "list")

	return NewInt(int64(len(vals[0].(*Cell).Values)))
}

// Vec
// =======================

func builtinVec(env *Environment, vals []Value) Value {
	return NewVector(vals)
}

func builtinVecToList(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)
	AssetArgType(vals[0], "vector")
	return NewCell(vals[0].(*Vector).Values)
}

// Math
// =======================

func builtinSmaller(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)
	AssetArgType(vals[0], "integer|float")
	AssetArgType(vals[1], "integer|float")

	var x, y float64
	switch v := vals[0].(type) {
	case Int:
		x = float64(v)
	case Float:
		x = float64(v)
	}
	switch v := vals[1].(type) {
	case Int:
		y = float64(v)
	case Float:
		y = float64(v)
	}

	return NewBool(x < y)
}

func valueAsFloat(v Value) float64 {
	switch v.Type() {
	case "integer":
		return float64(v.(Int))
	case "float":
		return float64(v.(Float))
	default:
		panic("valueAsFloat: non-number value given")
	}
}

func buildBuiltinOp(fn func(float64, float64) float64) func(*Environment, []Value) Value {
	return func(env *Environment, vals []Value) Value {
		AssetArgsSize(vals, 2, -1)
		AssetArgListTypes(NewCell(vals), "integer|float")

		var result float64 = valueAsFloat(vals[0])
		for _, v := range vals[1:] {
			result = fn(result, valueAsFloat(v))
		}

		if math.Trunc(result) == result {
			// Number is still integer
			return NewInt(int64(result))
		}

		return NewFloat(result)
	}
}

var builtinPlus = buildBuiltinOp(func(a, b float64) float64 {
	return a + b
})

var builtinMinus = buildBuiltinOp(func(a, b float64) float64 {
	return a - b
})

var builtinQuotient = buildBuiltinOp(func(a, b float64) float64 {
	return a / b
})

var builtinProduct = buildBuiltinOp(func(a, b float64) float64 {
	return a * b
})

var builtinMod = buildBuiltinOp(func(a, b float64) float64 {
	return float64(int64(a) % int64(b))
})

// Streams & I/O
// =======================

func builtinRead(env *Environment, vals []Value) Value {
	streamVal := env.Get("*in*")
	if len(vals) == 1 {
		streamVal = vals[0]
	}
	AssetArgType(streamVal, "stream")

	stream := streamVal.(*Stream)

	if stream.Direction != StreamDirIn {
		panic("read: given stream with direction not equal to 'in'")
	}

	r, _, err := stream.In.ReadRune()
	if err != nil {
		panic(fmt.Sprintf("read: %s", err))
	}

	return NewString(string(r))
}

func builtinWrite(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 2)

	streamVal := env.Get("*out*")
	stringVal := vals[0]

	if len(vals) == 2 {
		streamVal = vals[0]
		stringVal = vals[1]
	}

	AssetArgType(streamVal, "stream")
	AssetArgType(stringVal, "string")

	stream := streamVal.(*Stream)
	if stream.Direction != StreamDirOut {
		panic("write: given stream with direction not equal to 'out'")
	}

	_, err := stream.Out.Write([]byte(stringVal.(String)))
	if err != nil {
		panic(fmt.Sprintf("write: %s", err))
	}
	err = stream.Out.Flush()
	if err != nil {
		panic(fmt.Sprintf("write: %s", err))
	}

	return stringVal
}

// OS
// =======================

func builtinExit(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 0, 1)

	exitCode := 0
	if len(vals) == 1 {
		AssetArgType(vals[0], "integer")
		exitCode = int(vals[0].(Int))
	}

	os.Exit(exitCode)
	return NULL
}
