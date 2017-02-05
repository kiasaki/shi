package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	. "github.com/shi-lang/shi"

	_ "net/http/pprof"

	_ "github.com/shi-lang/shi/lib/shi/http"
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
	go func() {
		log.Println(http.ListenAndServe("localhost:8080", nil))
	}()

	shiPaths := strings.Split(os.Getenv("SHI_PATH"), ":")
	if len(shiPaths) == 1 && shiPaths[0] == "" {
		goPathLib := os.Getenv("GOPATH") + "/src/github.com/shi-lang/shi/lib"
		shiPaths = []string{".", "./lib", "/usr/share/shi/lib", goPathLib}
	}

	defer func() {
		fmt.Printf("Untrapped error!\n\n")
		if r := recover(); r != nil {
			for i := len(EvalStack) - 1; i >= 0; i-- {
				fmt.Printf("%d> %s\n", i+1, EvalStack[i])
			}
		}
		os.Exit(1)
	}()

	env := NewRootEnvironment()
	env.Set("*version*", NewString(ShiVersion))
	env.Set("*args*", StringArrayToList(os.Args))
	env.Set("*shi-path*", StringArrayToList(shiPaths))

	BuiltinLoad(env, []Value{NewString("shi:core")})

	env.Set("*module*", NewSym("global"))

	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			Eval(env, ParseFile(arg))
		}
	} else if stat, err := os.Stdin.Stat(); err == nil && stat.Size() > 0 {
		run(env, "stdin", os.Stdin)
	} else {
		BuiltinLoad(env, []Value{NewString("shi:repl")})
		Eval(env, NewCell([]Value{NewSym("shi:repl:repl-run")}))
	}
}

func run(env *Environment, name string, input io.ReadWriteCloser) {
	// Read all input
	contents, err := ioutil.ReadAll(input)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	// Parse and eval all top level instruction
	toplevel := Parse(name, string(contents))
	for _, expr := range toplevel {
		Eval(env, expr)
	}
}
