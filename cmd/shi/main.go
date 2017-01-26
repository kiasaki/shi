package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/kiasaki/shi"
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

	BuiltinLoad(env, []Value{NewString("shi::core")})

	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			Eval(env, ParseFile(arg))
		}
	} else if stat, err := os.Stdin.Stat(); err == nil && stat.Size() > 0 {
		run(env, "stdin", os.Stdin)
	} else {
		BuiltinLoad(env, []Value{NewString("shi::repl")})
		Eval(env, Parse("repl", "(repl-run)")[0])
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
		Eval(env, expr)
	}
}
