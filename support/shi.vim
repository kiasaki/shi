" Vim syntax file
" Language:     Shi
" Maintainer:   Frederic Gingras <frederic@gingras.cc>
" Last change:  30 Jan 2017

if exists("b:current_syntax")
  finish
endif

syntax case match

" ! $ - / 0-9 < = > ? @ a-zA-Z _
setlocal iskeyword=33,35-37,42,43,45,47,48-57,60-64,124,@,_
syntax match shiSymbol "\<\k\+\>"

" builtin
syntax keyword shiBuiltin macro
syntax keyword shiBuiltin def
syntax keyword shiBuiltin set
syntax keyword shiBuiltin fn
syntax keyword shiBuiltin if cond
syntax keyword shiBuiltin do
syntax keyword shiBuiltin while
syntax keyword shiBuiltin defn defmacro defobj new super
syntax keyword shiBuiltin let
syntax keyword shiBuiltin self

" builtin operators
syntax keyword shiBuiltin "+"
syntax keyword shiBuiltin "-"
syntax keyword shiBuiltin "*"
syntax keyword shiBuiltin "/"
syntax keyword shiBuiltin mod
syntax keyword shiBuiltin "<"
syntax keyword shiBuiltin "<="
syntax keyword shiBuiltin ">"
syntax keyword shiBuiltin ">="
syntax keyword shiBuiltin "="
syntax keyword shiBuiltin "/="
syntax keyword shiBuiltin ":"

" stdlib shi
syntax keyword shiFunc quote gensym macro-expand
syntax keyword shiFunc apply
syntax keyword shiFunc list
syntax keyword shiFunc type
syntax keyword shiFunc true? nil? int? str? cons? list? sym? prim? atom? obj? fn? macro?
syntax keyword shiFunc apply identity compose curry ->
syntax keyword shiFunc box unbox swap!
syntax keyword shiFunc alist? alist-has? alist-get alist-set alist-set-in alist-update
syntax keyword shiFunc alist-update-in alist-del alist-keys alist-vals
syntax keyword shiFunc obj obj-get obj-set obj-del obj-proto obj-proto-set!
syntax keyword shiFunc cons car cdr set-car!
syntax keyword shiFunc first rest caar cadr second cdar cddr
syntax keyword shiFunc caaar caadr cadar caddr third cdaar cdadr cddar cdddr
syntax keyword shiFunc range pair? odd? add1 sub1 min max abs num->str
syntax keyword shiFunc length reverse nth empty?
syntax keyword shiFunc eq? eql? not when unless and or
syntax keyword shiFunc dolist dotimes map filter foreach reduce
syntax keyword shiFunc conj extend
syntax keyword shiFunc unquote unquote-splicing quasiquote
syntax keyword shiFunc pr-str write getenv open close readb writeb exit rand millis seconds
syntax keyword shiFunc newline pr prn print println
syntax keyword shiFunc sleep open close read bind-inet socket listen accept
syntax keyword shiFunc str str-len read-all
syntax keyword shiFunc error trap-error

syntax match shiStringEscape "\v\\%([\\btnfr"]|u\x{4}|[0-3]\o{2}|\o{1,2})" contained
syntax region shiString start=/"/ skip=/\\"/ end=/"/ contains=shiStringEscape,@Spell

syntax match shiNumber "\v<[-+]?%(0|[1-9]\d*)\.?\d*>"

syntax keyword shiBoolean nil t

syntax match shiVarArg "\."

syntax match shiComment ";.*$" contains=shiCommentTodo,@Spell
syntax keyword shiCommentTodo contained FIXME TODO HACK FIXME: TODO: HACK:

syntax cluster shiTop contains=@Spell,shiBracketError,shiComment,shiVarArg,shiBoolean,shiNumber,shiString,shiStringEscape,shiBuiltin,shiFunc,shiSymbol,shiList,shiArray,shiObject

syntax region shiList   matchgroup=shiDelim start="("  matchgroup=shiDelim end=")" contains=@shiTop fold
syntax region shiArray  matchgroup=shiDelim start="\[" matchgroup=shiDelim end="\]" contains=@shiTop fold
syntax region shiObject matchgroup=shiDelim start="{"  matchgroup=shiDelim end="}" contains=@shiTop fold

syntax match shiBracketError display ")"

syntax sync fromstart

highlight default link shiBuiltin      Keyword
highlight default link shiFunc         Identifier
highlight default link shiString       String
highlight default link shiStringEscape Character
highlight default link shiNumber       Number
highlight default link shiBoolean      Boolean
highlight default link shiVarArg       Special

highlight default link shiComment      Comment
highlight default link shiCommentTodo  Todo

highlight default link shiDelim        Delimiter
highlight default link shiBracketError Error

highlight link shiDelim Comment

let b:current_syntax = "shi"
