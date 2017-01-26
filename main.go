package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

const ShiVersion = "0.1.0"

func main() {
	env := NewRootEnvironment()
	LoadPrelude(env)

	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			runFile(env, arg)
		}
	}
	if stat, err := os.Stdin.Stat(); err == nil && stat.Size() > 0 {
		run(env, "stdin", os.Stdin)
	} else {
		shiRoot := os.Getenv("SHI_ROOT")
		if shiRoot == "" {
			shiRoot = "."
		}
		runFile(env, shiRoot+"/shi/repl.shi")
	}
}

func runFile(env *Environment, name string) {
	file, err := os.Open(name)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
	run(env, name, file)
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
