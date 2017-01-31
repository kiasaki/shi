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
syntax match ethSymbol "\<\k\+\>"

" builtin
syntax keyword shiBuiltin fn
syntax keyword shiBuiltin do
syntax keyword shiBuiltin cond
syntax keyword shiBuiltin loop
syntax keyword shiBuiltin recur
syntax keyword shiBuiltin quote
syntax keyword shiBuiltin macro
syntax keyword shiBuiltin type
syntax keyword shiBuiltin parse
syntax keyword shiBuiltin eval
syntax keyword shiBuiltin load
syntax keyword shiBuiltin error
syntax keyword shiBuiltin trap-error
syntax keyword shiBuiltin environment
syntax keyword shiBuiltin root-environment
syntax keyword shiBuiltin environment-get
syntax keyword shiBuiltin environment-set
syntax keyword shiBuiltin environment-root
syntax keyword shiBuiltin eq
syntax keyword shiBuiltin eql
syntax keyword shiBuiltin sym
syntax keyword shiBuiltin str
syntax keyword shiBuiltin str-nth
syntax keyword shiBuiltin str-join
syntax keyword shiBuiltin str-slice
syntax keyword shiBuiltin str-length
syntax keyword shiBuiltin str-split
syntax keyword shiBuiltin list
syntax keyword shiBuiltin list-nth
syntax keyword shiBuiltin list-join
syntax keyword shiBuiltin list-slice
syntax keyword shiBuiltin list-length
syntax keyword shiBuiltin vec
syntax keyword shiBuiltin vec-nth
syntax keyword shiBuiltin vec-join
syntax keyword shiBuiltin vec-slice
syntax keyword shiBuiltin vec-length
syntax keyword shiBuiltin empty-map
syntax keyword shiBuiltin map-get
syntax keyword shiBuiltin map-set
syntax keyword shiBuiltin map-delete
syntax keyword shiBuiltin map-keys
syntax keyword shiBuiltin read
syntax keyword shiBuiltin write
syntax keyword shiBuiltin open
syntax keyword shiBuiltin close
syntax keyword shiBuiltin exit

" builtin functions
syntax keyword shiBuiltin get
syntax keyword shiBuiltin set
syntax keyword shiBuiltin def
syntax keyword shiBuiltin defn
syntax keyword shiBuiltin defmacro
syntax keyword shiBuiltin let
syntax keyword shiBuiltin if
syntax keyword shiBuiltin or
syntax keyword shiBuiltin and

syntax keyword shiBuiltin module
syntax keyword shiBuiltin import

" builtin operators
syntax keyword shiBuiltin "+"
syntax keyword shiBuiltin "-"
syntax keyword shiBuiltin "*"
syntax keyword shiBuiltin "/"
syntax keyword shiBuiltin "%"
syntax keyword shiBuiltin "<"
syntax keyword shiBuiltin "<="
syntax keyword shiBuiltin ">"
syntax keyword shiBuiltin ">="
syntax keyword shiBuiltin "=="
syntax keyword shiBuiltin "!="

" stdlib shi
syntax keyword shiFunc gensym unquote unquote-splicing quasiquote
syntax keyword shiFunc read-line
syntax keyword shiFunc newline
syntax keyword shiFunc pr
syntax keyword shiFunc prn
syntax keyword shiFunc print
syntax keyword shiFunc println
syntax keyword shiFunc null? bool? true? false? list? vec? sym? int? float? stream? map? builtin? closure? macro? environment?
syntax keyword shiFunc empty?
syntax keyword shiFunc first second head tail length append
syntax keyword shiFunc identity
syntax keyword shiFunc not

syntax match shiKeyword ":\<\k\+\>"

syntax match shiStringEscape "\v\\%([\\btnfr"]|u\x{4}|[0-3]\o{2}|\o{1,2})" contained
syntax region shiString start=/"/ skip=/\\"/ end=/"/ contains=shiStringEscape,@Spell

syntax match shiNumber "\v<[-+]?%(0|[1-9]\d*)\.?\d*>"

syntax keyword shiBoolean null true false

syntax match shiVarArg "&"

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
