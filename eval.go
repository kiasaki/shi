package shi

import (
	"fmt"
	"os"
)

func EvalList(env *Environment, v *Cell) Value {
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

	if c, ok := head.(Callable); ok {
		return c.Call(env, v.Values[1:])
	} else {
		panic(fmt.Sprintf("trying to call non-callable value `%s'", head.String()))
	}
}

func Eval(env *Environment, v Value) Value {
	switch t := v.(type) {
	case *Cell:
		return EvalList(env, t)
	default:
		return t
	}
}

func FullEval(env *Environment, v Value) Value {
	if v.Type() == "symbol" {
		if vv := env.Get(v.String()); vv != NULL {
			return vv
		}
	}
	return Eval(env, v)
}

func buildCallEnv(doEval bool, evalEnv, argsEnv *Environment, wantedArgs []string, args []Value) []string {
	rest := ""
	restValues := []Value{}
	argNames := wantedArgs

	var eval = func(arg Value) Value {
		if doEval {
			return FullEval(argsEnv, arg)
		}
		return arg
	}

	for _, arg := range args {
		if len(argNames) == 0 && rest == "" {
			panic(fmt.Sprintf(
				"function called with too many arguments: wanted %v, got %v. Args: %v",
				len(wantedArgs), len(args), NewCell(args),
			))
		} else if rest != "" {
			restValues = append(restValues, eval(arg))
		} else if argNames[0] == "&" {
			if len(argNames) != 2 {
				panic(fmt.Sprintf("found illegal '&' in argument list: %s", NewCell(args)))
			}
			rest = argNames[1]
			argNames = []string{}
			restValues = append(restValues, eval(arg))
		} else {
			evalEnv.Set(argNames[0], eval(arg))
			argNames = argNames[1:]
		}
	}

	if rest != "" {
		evalEnv.Set(rest, NewCell(restValues))
	}

	return argNames
}

func macroQuote(val Value) Value {
	switch v := val.(type) {
	case *Cell, *Sym:
		return NewCell([]Value{NewSym("quote"), val})
	default:
		return v
	}
}

func macroQQ(env *Environment, val Value, level int) Value {
	switch v := val.(type) {
	case *Cell:
		if len(v.Values) == 0 {
			return nil
		}
		if v.Values[0].String() == "unquote" && level == 0 {
			return v.Values[1]
		}
		appendArgs := macroQQAppendArgs(val, level)
		return BuildCall("list-join", appendArgs)
	default:
		return macroQuote(v)
	}
}

func macroQQAppendArgs(val Value, level int) []Value {
	if v, ok := val.(*Cell); ok {
		if len(v.Values) == 0 {
			return v
		}
		switch v.Values[0].String() {
		case "unquote":
			if level == 0 {
				return v.Values[1:]
			}
			level--
		case "quasiquote":
			level++
		}
		a := macroQQAppendArg(v.Values[0])
	}
	return []Value{macroQuote(val)}
}
