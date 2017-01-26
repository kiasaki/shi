package main

import (
	"fmt"
	"io"
	"math"
	"os"
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
	AddBuiltinSpecial(env, "quote", builtinQuote)
	AddBuiltinSpecial(env, "fn", builtinFn)
	AddBuiltinSpecial(env, "apply", builtinApply)
	AddBuiltin(env, "do", builtinDo)
	AddBuiltin(env, "error", builtinError)
	AddBuiltin(env, "trap-error", builtinTrapError)

	AddBuiltinSpecial(env, "set", builtinSet)
	AddBuiltin(env, "root-environment", builtinRootEnvironment)
	AddBuiltinSpecial(env, "eval", builtinEval)
	AddBuiltin(env, "parse", builtinParse)

	env.Set(builtinValStr.Name, builtinValStr)
	env.Set(builtinValStrJoin.Name, builtinValStrJoin)

	AddBuiltin(env, "+", builtinPlus)
	AddBuiltin(env, "-", builtinMinus)
	AddBuiltin(env, "/", builtinQuotient)
	AddBuiltin(env, "*", builtinProduct)

	env.Set(builtinValList.Name, builtinValList)

	AddBuiltin(env, "write", builtinWrite)
	AddBuiltin(env, "read-line", builtinReadLine)
	AddBuiltin(env, "print-str", builtinPrintStr)

	env.Set("*stdin*", NewStream(os.Stdin))
	env.Set("*stdout*", NewStream(os.Stdout))
	env.Set("*stderr*", NewStream(os.Stderr))
	env.Set("*version*", NewString(ShiVersion))
}

// Language
// =======================

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

func builtinSet(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)
	AssetArgType(vals[0], "symbol")

	valueToSet := FullEval(env, vals[1])
	env.Set(vals[0].String(), valueToSet)
	return valueToSet
}

func builtinRootEnvironment(env *Environment, vals []Value) Value {
	return NewRootEnvironment()
}

func builtinEval(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)

	e := FullEval(env, vals[0])
	AssetArgType(e, "environment")

	return vals[1].Eval(e.(*Environment))
}

func builtinParse(env *Environment, vals []Value) Value {
	AssetArgsSize(vals, 2, 2)
	AssetArgType(vals[0], "string")
	AssetArgType(vals[1], "string")

	topLevel := Parse(string(vals[0].(String)), string(vals[1].(String)))

	if len(topLevel) == 1 {
		return topLevel[0]
	} else {
		// Wrap topLevel in a do so it can be passed as a value
		return NewCell(append([]Value{NewSym("do")}, topLevel...))
	}
}

// String
// =======================

func builtinStr(env *Environment, vals []Value) Value {
	AssetArgListType(NewCell(vals), "string")
	return builtinStrJoin(env, []Value{NewString(""), NewCell(vals)})
}

var builtinValStr = NewBuiltin("str", BuiltinFn(builtinStr)).(*Builtin)

func builtinStrJoin(env *Environment, vals []Value) Value {
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

var builtinValStrJoin = NewBuiltin("str-join", BuiltinFn(builtinStrJoin)).(*Builtin)

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

var builtinValList = NewBuiltin("list", BuiltinFn(builtinList)).(*Builtin)

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
