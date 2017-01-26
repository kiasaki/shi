package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func AddBuiltin(env *Environment, name string, fn func(*Environment, []Value) Value) {
	env.Set(name, NewBuiltin(name, BuiltinFn(fn)))
}

func AddBuiltinSpecial(env *Environment, name string, fn func(*Environment, []Value) Value) {
	b := NewBuiltin(name, BuiltinFn(fn))
	b.(*Builtin).IsSpecial = true
	env.Set(name, b)
}

func AddBuiltins(env *Environment) {
	AddBuiltinSpecial(env, "type", builtinType)
	AddBuiltinSpecial(env, "quote", builtinQuote)
	AddBuiltinSpecial(env, "fn", builtinFn)
	AddBuiltinSpecial(env, "apply", builtinApply)
	AddBuiltin(env, "do", builtinDo)

	AddBuiltin(env, "error", builtinError)
	AddBuiltin(env, "trap-error", builtinTrapError)

	AddBuiltin(env, "environment", builtinEnvironment)
	AddBuiltin(env, "root-environment", builtinRootEnvironment)
	AddBuiltinSpecial(env, "environment-set", builtinEnvironmentSet)
	AddBuiltinSpecial(env, "environment-get", builtinEnvironmentGet)
	AddBuiltin(env, "environment-root", builtinEnvironmentRoot)
	AddBuiltinSpecial(env, "eval", builtinEval)
	AddBuiltin(env, "parse", builtinParse)
	AddBuiltin(env, "parse-file", builtinParseFile)
	AddBuiltin(env, "load", builtinLoad)

	AddBuiltin(env, "sym", builtinSym)

	AddBuiltin(env, "str", builtinStr)
	AddBuiltin(env, "str-join", builtinStrJoin)

	AddBuiltin(env, "+", builtinPlus)
	AddBuiltin(env, "-", builtinMinus)
	AddBuiltin(env, "/", builtinQuotient)
	AddBuiltin(env, "*", builtinProduct)

	AddBuiltin(env, "list", builtinList)

	AddBuiltin(env, "write", builtinWrite)
	AddBuiltin(env, "read-line", builtinReadLine)
	AddBuiltin(env, "print-str", builtinPrintStr)

	env.Set("*stdin*", NewStream(os.Stdin))
	env.Set("*stdout*", NewStream(os.Stdout))
	env.Set("*stderr*", NewStream(os.Stderr))
}

// Language
// =======================

func builtinType(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)

	return NewString(vals[0].Type())
}

func builtinQuote(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)

	return vals[0]
}

func builtinFn(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, -1)
	AssetArgType(vals[0], "list")
	AssetArgListType(vals[0], "symbol")

	argNames := []string{}
	for _, arg := range vals[0].(*Cell).Values {
		argNames = append(argNames, arg.String())
	}
	return NewClosure(env, argNames, vals[1:])
}

func builtinApply(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)

	args := FullEval(env, vals[1])
	AssetArgType(args, "list")

	return NewCell(append([]Value{vals[0]}, args.(*Cell).Values...)).Eval(env)
}

func builtinDo(env *Environment, vals []Value) Value {
	if len(vals) == 0 {
		return NULL
	} else {
		return vals[len(vals)-1]
	}
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
			ret = NewCell([]Value{vals[1], NewString(fmt.Sprintf("%v", r))}).Eval(env)
		}
	}()

	ret = NewCell([]Value{vals[0]}).Eval(env)
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
		givenEnv := vals[0].Eval(env)
		AssetArgType(givenEnv, "environment")
		e = givenEnv.(*Environment)
		k = vals[1]
		v = vals[2]
	}
	AssetArgType(k, "symbol")

	valueToSet := FullEval(env, v)
	e.Set(k.String(), valueToSet)
	return valueToSet
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

	return e.Get(k.String())
}

func builtinEnvironmentRoot(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 0, 1)
	e := env
	if len(vals) > 0 && vals[0].Type() == "environment" {
		e = vals[0].(*Environment)
	}

	return e.Root()
}

func builtinEval(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)

	e := FullEval(env, vals[0])
	AssetArgType(e, "environment")

	return vals[1].Eval(e.(*Environment))
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

	topLevel := Parse(string(name.(String)), string(name.(String)))

	if len(topLevel) == 1 {
		return topLevel[0]
	} else {
		// Wrap topLevel in a do so it can be passed as a value
		return NewCell(append([]Value{NewSym("do")}, topLevel...))
	}
}

func builtinParseFile(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)
	AssetArgType(vals[0], "string")

	return ParseFile(string(vals[0].(String)))
}

func builtinLoadHelper(env *Environment, file string) bool {
	targetFile := file
	if len(targetFile) < 4 || targetFile[len(targetFile)-4:] != ".shi" {
		// Ensure extension is present
		targetFile = targetFile + ".shi"
	}

	targetModule := filepath.Join(file, filepath.Base(targetFile))

	if _, err := os.Stat(targetFile); err == nil {
		// Try file
		ParseFile(targetFile).Eval(env)
		return true
	} else if _, err := os.Stat(targetModule); err == nil {
		// Try module
		ParseFile(targetModule).Eval(env)
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
	shiPathsVal := env.Get("*shi-path*").Eval(env)
	if shiPathsVal.Type() == "cell" {
		panic("load: expected '*shi-path*' to be a string list")
	}
	shiPaths := shiPathsVal.(*Cell).Values
	targetModule := strings.Replace(targetFile, "::", string(os.PathSeparator), -1)

	for _, p := range shiPaths {
		if builtinLoadHelper(env, filepath.Join(string(p.(String)), targetModule)) {
			return NULL
		}
	}

	panic(fmt.Sprintf("load: could not find module '%s'", targetFile))
}

// Symbol
// =======================

func builtinSym(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)
	AssetArgType(vals[0], "string")

	return NewSym(string(vals[0].(String)))
}

// String
// =======================

func builtinStr(env *Environment, vals []Value) Value {
	AssetArgListType(NewCell(vals), "string")
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

// Numbers
// =======================

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

// List
// =======================

func builtinList(env *Environment, vals []Value) Value {
	return NewCell(vals)
}

// Streams & I/O
// =======================

func builtinWrite(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)
	AssetArgType(vals[0], "stream")
	AssetArgType(vals[1], "string")

	vals[0].(*Stream).Write([]byte(vals[1].(String)))
	return vals[1]
}

func builtinReadLine(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 1, 1)
	AssetArgType(vals[0], "stream")

	var err error
	var bs = []byte{}
	var b = make([]byte, 1)
	for b[0] != byte('\n') {
		_, err = vals[0].(*Stream).Value.Read(b)
		if err == io.EOF {
			if len(bs) == 0 {
				return NULL
			} else {
				return NewString(string(bs))
			}
		} else if err != nil {
			panic(fmt.Sprintf("read-line: %v", err))
		}
		bs = append(bs, b[0])
	}

	return NewString(string(bs))
}

func builtinPrintStr(env *Environment, vals []Value) Value {
	pretty := NewCell(vals).String()
	return NewString(pretty[1 : len(pretty)-1])
}
