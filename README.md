# Shi Lisp (シ)

Built-in

```
*version*
*args*
*shi-path*
*stdin*
*stdout*
*stderr*
*filename*
*dirname*

(type val)
(quote val) !todo!move-to-core
(fn (args...) body...)
(apply fn args)
(do expr...) !todo!move-to-core

(error message)
(trap-error try-fn catch-fn)

(environment parent)
(root-environment)
(environment-set *env*? k v)
(environment-get *env*? k)
(environment-root *env*)
(eval env val)
(parse name? contents)
(parse-file file-name) !todo!move-to-core
(load file-name-or-module) !todo!move-to-core

(sym str)

(str vals...)
(str-join sep strs)

(+ x y)
(- x y)
(/ x y)
(* x y)

(list vals...) !todo!move-to-core

(write stream string) !todo!move-to-core
(read-line stream) !todo!move-to-core
(print-str val)
```

Core

```
*in*
*out*
*macros*
*packages*
*package*

set-global
set
value

print
println
```
