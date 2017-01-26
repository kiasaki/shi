package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const ShiVersion = "0.1.0"

func StringArrayToList(vs []string) Value {
	list := []Value{NewSym("list")}
	for _, v := range vs {
		list = append(list, NewString(v))
	}
	return NewCell(list)
}

func main() {
	shiPaths := strings.Split(os.Getenv("SHI_PATH"), ":")
	if len(shiPaths) == 1 && shiPaths[0] == "" {
		shiPaths = []string{".", "./lib", "/usr/share/shi"}
	}

	env := NewRootEnvironment()
	env.Set("*version*", NewString(ShiVersion))
	env.Set("*args*", StringArrayToList(os.Args))
	env.Set("*shi-path*", StringArrayToList(shiPaths))

	builtinLoad(env, []Value{NewString("shi::core")})

	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			ParseFile(arg).Eval(env)
		}
	} else if stat, err := os.Stdin.Stat(); err == nil && stat.Size() > 0 {
		run(env, "stdin", os.Stdin)
	} else {
		builtinLoad(env, []Value{NewString("shi::repl")})
		Parse("repl", "(repl-run)")[0].Eval(env)
	}
}

func run(env *Environment, name string, input io.ReadWriteCloser) {
	// Read all input
	contents, err := ioutil.ReadAll(input)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
	input.Close()

	// Parse and eval all top level instruction
	toplevel := Parse(name, string(contents))
	for _, expr := range toplevel {
		expr.Eval(env)
	}
}
