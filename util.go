package shi

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
	if v == nil {
		panic(fmt.Sprintf("got unbound value, expected %s", typeName))
	}
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

func BuildCall(symName string, vals []Value) Value {
	return NewCell(append([]Value{NewSym(symName)}, vals...))
}

func printValues(ld, rd string, readably bool, vals []Value) string {
	formatted := ld
	for i, value := range vals {
		if i > 0 {
			formatted += " "
		}
		if v, ok := value.(ReadablyStringer); ok && readably {
			formatted += v.ReadableString()
		} else {
			formatted += value.String()
		}
	}
	return formatted + rd
}

func groupValsAsPairs(receiver string, vals []Value) [][]Value {
	if len(vals)%2 != 0 {
		panic(fmt.Sprintf("%s: wanted a pair number of values, got %d", receiver, len(vals)))
	}

	pairs := [][]Value{}
	for i := 0; i < len(vals); i += 2 {
		pairs = append(pairs, vals[i:i+2])
	}
	return pairs
}
