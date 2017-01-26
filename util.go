package main

import (
	"fmt"
	"strings"
)

func AssetArgsSize(vals []Value, min int, max int) {
	if min == max && len(vals) != min {
		panic(fmt.Sprintf("expected exactly '%v' arguments, got '%v'", min, len(vals)))
	}
	if len(vals) < min {
		panic(fmt.Sprintf("expected minimum '%v' arguments, got '%v'", min, len(vals)))
	}
	if len(vals) > max && max != -1 {
		panic(fmt.Sprintf("expected maximum '%v' arguments, got '%v'", max, len(vals)))
	}
}

func AssetArgType(v Value, typeName string) {
	if v.Type() != typeName {
		panic(fmt.Sprintf(
			"expected argument to be of type '%s', got '%s' in `%s'",
			typeName, v.Type(), v.String(),
		))
	}
}

func AssetArgListType(v Value, typeName string) {
	for _, v := range v.(*Cell).Values {
		if v.Type() != typeName {
			panic(fmt.Sprintf(
				"expected argument to be of type '%s', got '%s' in `%s'",
				typeName, v.Type(), v.String(),
			))
		}
	}
}

func AssetArgListTypes(v Value, typeNames string) {
	for _, v := range v.(*Cell).Values {
		if strings.Contains(v.Type(), typeNames) {
			panic(fmt.Sprintf(
				"expected argument to be of type '%s', got '%s' in `%s'",
				typeNames, v.Type(), v.String(),
			))
		}
	}
}

func FullEval(env *Environment, arg Value) Value {
	v := arg.Eval(env)
	if v.Type() == "symbol" {
		if vv := env.Get(v.String()); vv != NULL {
			v = vv
		}
	}
	return v
}
