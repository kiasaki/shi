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
syntax keyword shiBuiltin defn defmacro
syntax keyword shiBuiltin let

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

" stdlib shi
syntax keyword shiFunc quote gensym macro-expand
syntax keyword shiFunc apply
syntax keyword shiFunc list
syntax keyword shiFunc type
syntax keyword shiFunc true? nil? int? str? cons? list? sym? prim? atom?
syntax keyword shiFunc apply call identity
syntax keyword shiFunc cons car cdr set-car!
syntax keyword shiFunc first rest caar cadr second cdar cddr
syntax keyword shiFunc caaar caadr cadar caddr third cdaar cdadr cddar cdddr
syntax keyword shiFunc length reverse nth empty?
syntax keyword shiFunc eq? eql? not when unless and or
syntax keyword shiFunc unquote unquote-splicing quasiquote
syntax keyword shiFunc pr-str write getenv open close readb writeb exit rand millis seconds
syntax keyword shiFunc newline pr prn print println
syntax keyword shiFunc sleep open close read bind-inet socket listen accept
syntax keyword shiFunc str str-len read-all

syntax match shiKeyword ":\<\k\+\>"

syntax match shiStringEscape "\v\\%([\\btnfr"]|u\x{4}|[0-3]\o{2}|\o{1,2})" contained
syntax region shiString start=/"/ skip=/\\"/ end=/"/ contains=shiStringEscape,@Spell

syntax match shiNumber "\v<[-+]?%(0|[1-9]\d*)\.?\d*>"

syntax keyword shiBoolean nil t

syntax match shiVarArg "\."

syntax match shiComment ";.*$" contains=shiCommentTodo,@Spell
syntax keyword shiCommentTodo contained FIXME TODO HACK FIXME: TODO: HACK:

syntax cluster shiTop contains=@Spell,shiBracketError,shiComment,shiVarArg,shiBoolean,shiNumber,shiKeyword,shiString,shiStringEscape,shiBuiltin,shiFunc,shiSymbol,shiList,shiArray,shiObject

syntax region shiList   matchgroup=shiDelim start="("  matchgroup=shiDelim end=")" contains=@shiTop fold
syntax region shiArray  matchgroup=shiDelim start="\[" matchgroup=shiDelim end="\]" contains=@shiTop fold
syntax region shiObject matchgroup=shiDelim start="{"  matchgroup=shiDelim end="}" contains=@shiTop fold

syntax match shiBracketError display ")"

syntax sync fromstart

highlight default link shiBuiltin      Keyword
highlight default link shiFunc         Identifier
highlight default link shiKeyword      String
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
