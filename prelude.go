package main

const PRELUDE = `
(set *out* *stdout*)

(set print (fn (& v)
  (write *out* (apply print-str v))))

(set println (fn (& v)
  (write *out* (apply print-str v))
  (write *out* "\n")))

`

func LoadPrelude(env *Environment) {
	toplevel := Parse("prelude", PRELUDE)
	for _, expr := range toplevel {
		expr.Eval(env)
	}
}
