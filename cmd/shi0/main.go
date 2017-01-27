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

	for _, expr := range Parse("prelude", Prelude) {
		Eval(env, expr)
	}

	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			Eval(env, ParseFile(arg))
		}
	} else if stat, err := os.Stdin.Stat(); err == nil && stat.Size() > 0 {
		run(env, "stdin", os.Stdin)
	} else {
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

var Prelude = strings.Replace(`
; Shi0 Prelude

; Required built-ins
;
; - fn
; - do
; - cond
; - loop / recur
; - quote
; - macro
;
; - pr-str
; - type
; - read
; - eval
; - load
;
; - error
; - trap-error
;
; - environment
; - root-environment
; - environment-get
; - environment-set
; - environment-root
;
; - eq
; - eql
; - sym
; - str
; - str-nth
; - str-join
; - str-slice
; - list
; - list-nth
; - list-join
; - list-slice
; - vec
; - vec-nth
; - vec-join
; - vec-slice
;
; - +
; - -
; - /
; - *
; - mod
;
; - read
; - write
; - *stdin*
; - *stdout*
; - *stderr*

(environment-set def (macro (name value)
  (list 'environment-set '(environment-root) name value)))

(environment-set set! (macro (name value)
  (list 'environment-set name value)))

(def defn (macro (name args & exprs)
  (list 'def name (cons 'fn (cons args exprs)))))

(def defmacro (macro (name args & exprs)
  (list 'def name (cons 'macro (cons args exprs)))))

(defmacro if (test then & else)
  ~(cond (,test ,then)
         ,@(cond (else ~((true ,@else))))))

(defmacro when (test & body)
  ~(cond (,test ,@body)))

(defmacro let (args & body)
  ((fn (vars vals)
     (defn vars (x)
       (cond (x (cons (if (atom (car x))
                          (car x)
                        (caar x))
                      (vars (cdr x))))))
     (defn vals (x)
       (cond (x (cons (if (atom (car x))
                          nil
                        (cadar x))
                      (vals (cdr x))))))
     ~((lambda ,(vars args) ,@body) ,@(vals args)))
   nil nil))

(def *in* *stdin*)
(def *out* *stdout*)

(defn read-line ()
  (loop (line "")
    (def c (read))
    (cond
      (eq c "\n") line
      else (recur (str line c)))))

(defn newline ()
  (write "\n"))

(defn pr (v)
  (write (pr-str v)))

(defn prn (v)
  (pr v)
  (newline))

(defn print (v)
  (set! *print-readably* false)
  (write (pr-str v)))

(defn println (v)
  (print v)
  (newline))

(defn repl (env)
  (write "> ")

  (trap-error
    (fn ()
      (println (eval env (parse (read-line)))))
    (fn (e)
      (prn e)))

  (repl env))

(defn repl-run ()
  (write (str "Shi Lisp v" *version* " REPL\n"))
  (def env (root-environment))
  (repl env))
`, "~", "`", -1)
