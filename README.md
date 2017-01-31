# Shi Programming Language (シ)

_A productive and simple language for rapid exploration and iteration._

## What's Shi

Shi is an interpreted language that embraces the functional programming paradigm. It has many
features to help you quickly write websites, APIs and system management scripts. It's easy to work
with and helps you quickly get from start to finish.

## Features

- Simple syntax
- Functions are first class citizens
- Large standard library
- Package manager and good project tooling
- Garbage collected
- Macros
- Currying on all functions
- Immutable datastructures
- More functions, less datatypes: null, bool, string, int, float, list, vector, map, stream

## Status

Shi is currently in Alpha. It's working for some use cases but we are still working on defining
a good standard library, creating amazing tooling and documenting the language.

## Installing

Given that you have `go` install, an easy way to get `shi` installed is to run

```
go get github.com/shi-lang/shi
go install github.com/shi-lang/shi/cmd/shi
```

## Documentation

Visit [https://www.shi-lang.org/docs/](https://www.shi-lang.org/docs/).

## Building

Shi's interpreter is build using Golang, so, provided you have Go installed you should be able
to run:

```
make build
```

This will build the `shi` tool. To test it out simply run `./shi`.

The next step is to try and modify one of the `.go` files or some of the standard library `.shi`
files and run `make build` / `make test` again.

## Community

## Contributing

Read the general [Contributing guide](https://github.com/shi-lang/shi/blob/master/CONTRIBUTING.md), and then:

- Fork it ([https://github.com/shi-lang/shi/fork](https://github.com/shi-lang/shi/fork))
- Create your feature branch (`git checkout -b my-new-feature`)
- Commit your changes (`git commit -am 'Add some feature'`)
- Push to the branch (`git push origin my-new-feature`)
- Create a new Pull Request

## License

MIT. See `LICENSE` file.

```
*version*
*args*
*shi-path*
*filename*
*dirname*

; Language
fn
do
cond
loop
recur
quote
macro

; Basics
*print-readably*
pr-str
type
parse
eval
load

; Errors
error
trap-error

; Environments
environment
root-environment
environment-set
environment-get
environment-root

; Compare
eq
eql

; Symbols
sym

; Strings
str
str-join

; Lists
cons
list
list-nth
list-join
list-slice
list-length

; Math
+
-
/
*
mod

; Streams / IO
read
write
*stdin*
*stdout*
*stderr*

; OS
exit

; Prelude
def
set!
defn
defmacro
if
when
let
*in*
*out*
read-line
newline
pr
prn
print
println

repl
repl-run
```
