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

	head := Eval(env, v.Values[0])
	if c, ok := head.(Callable); ok {
		return c.Call(env, v.Values[1:])
	} else {
		panic(fmt.Sprintf("trying to call non-callable value `%s'", head.String()))
	}
}

var EvalStack = []Value{}

func Eval(env *Environment, v Value) Value {
	EvalStack = append(EvalStack, v)
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
		EvalStack = EvalStack[:len(EvalStack)-1]
	}()

	switch t := v.(type) {
	case *Sym:
		symValue := env.Get(t.Name)
		if symValue == nil {
			panic(fmt.Sprintf("eval: unbound symbol '%s'", t.Name))
		}
		return symValue
	case *Cell:
		return EvalList(env, t)
	case *Vector:
		vals := []Value{}
		for _, v := range t.Values {
			vals = append(vals, Eval(env, v))
		}
		return NewVector(vals)
	default:
		return t
	}
}

func buildCallEnv(doEval bool, callerEnv, argsEnv *Environment, wantedArgs []string, args []Value) []string {
	rest := ""
	restValues := []Value{}
	argNames := wantedArgs

	var eval = func(arg Value) Value {
		if doEval {
			return Eval(callerEnv, arg)
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
			argsEnv.Set(argNames[0], eval(arg))
			argNames = argNames[1:]
		}
	}

	if len(argNames) == 2 && argNames[0] == "&" {
		// there is a rest arg but not enough args for it
		// still mark as complete and pass an empty list
		rest = argNames[1]
		argNames = []string{}
	}

	if rest != "" {
		argsEnv.Set(rest, NewCell(restValues))
	}

	return argNames
}
