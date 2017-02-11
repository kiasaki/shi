#!/bin/bash

filter="$1"

function fail() {
  printf '\e[1;31m[ERROR]\e[0m '
  echo "$1"
  exit 1
}

function do_run() {
  error=$(echo "$3" | ./shi 2>&1 > /dev/null)
  if [ -n "$error" ]; then
    echo FAILED
    fail "$error"
  fi

  result=$(echo "(prn (do 
  $3
  ))" | ./shi 2> /dev/null | tail -1)
  if [ "$result" != "$2" ]; then
    echo FAILED
    fail "$2 expected, but got $result"
  fi
}

function run() {
  if [[ "$1" =~ "$filter" ]]; then
    echo -n "Testing $1 ... "
    # Run the tests twice to test the garbage collector with different settings.
    MINILISP_ALWAYS_GC= do_run "$@"
    MINILISP_ALWAYS_GC=1 do_run "$@"
    echo ok
  fi
}

# Basic data types
run integer 1 1
run integer -1 -1
run symbol a "'a"
run quote a "(quote a)"
run quote 63 "'63"
run quote '(+ 1 2)' "'(+ 1 2)"

run + 3 '(+ 1 2)'
run + -2 '(+ 1 -3)'

run 'unary -' -3 '(- 3)'
run '-' -2 '(- 3 5)'
run '-' -9 '(- 3 5 7)'

run '<' t '(< 2 3)'
run '<' '()' '(< 3 3)'
run '<' '()' '(< 4 3)'

run 'literal list' '(a b c)' "'(a b c)"
run 'literal list' '(a b . c)' "'(a b . c)"

# List manipulation
run cons "(a . b)" "(cons 'a 'b)"
run cons "(a b c)" "(cons 'a (cons 'b (cons 'c ())))"

run car a "(car '(a b c))"
run cdr "(b c)" "(cdr '(a b c))"

run set-car! "(x . b)" "(def obj (cons 'a 'b)) (set-car! obj 'x) obj"

# Comments
run comment 5 "
  ; 2
  5 ; 3"

# Global variables
run def 7 '(def x 7) x'
run def 10 '(def x 7) (+ x 3)'
run def 7 '(def + 7) +'
run set 11 '(def x 7) (set x 11) x'
run set 17 '(set + 17) +'

# Conditionals
run if1 a "(if 1 'a)"
run if2 '()' "(if () 'a)"
run if3 a "(if 1 'a 'b)"
run if4 a "(if 0 'a 'b)"
run if5 a "(if 'x 'a 'b)"
run if6 b "(if () 'a 'b)"
run if7 c "(if () 'a 'b 'c)"
run if8 d "(if () 'a () 'c 'd)"
run if9 '()' "(if () 'a () 'b () 'c)"

# Numeric comparisons
run = t '(= 3 3)'
run = '()' '(= 3 2)'

# eq?
run eq? t "(eq? 'foo 'foo)"
run eq? t "(eq? + +)"
run eq? '()' "(eq? 'foo 'bar)"
run eq? '()' "(eq? + 'bar)"

# gensym
run gensym G__0 '(gensym)'
run gensym '()' "(eq? (gensym) 'G__0)"
run gensym '()' '(eq? (gensym) (gensym))'
run gensym t '((fn (x) (eq? x x)) (gensym))'

# functions
run fn '<function>' '(fn (x) x)'
run fn t '((fn () t))'
run fn 9 '((fn (x) (+ x x x)) 3)'
run fn '(1 2 3)' '((fn xs xs) 1 2 3)'

run args 15 '(def f (fn (x y z) (+ x y z))) (f 3 5 7)'

run restargs '(3 5 7)' '(def f (fn (x . y) (cons x y))) (f 3 5 7)'
run restargs '(3)'    '(def f (fn (x . y) (cons x y))) (f 3)'

run do1 '()' '(do)'
run do2 '1' '(do 1)'
run do3 '2' '(do 1 2)'

# lexical closures
run closure 3 '(def call (fn (f) ((fn (var) (f)) 5)))
  ((fn (var) (call (fn () var))) 3)'

run counter 3 '
  (def counter
    ((fn (val)
       (fn () (set val (+ val 1)) val))
     0))
  (counter)
  (counter)
  (counter)'

# while
run while 45 "
  (def i 0)
  (def sum 0)
  (while (< i 10)
    (set sum (+ sum i))
    (set i (+ i 1)))
  sum"

# macro
run macro 42 "
  (def list (fn (x . y) (cons x y)))
  (def if-zero (macro (x then) (list 'if (list '= x 0) then)))
  (if-zero 0 42)"

run macro 7 '(def seven (macro () 7)) ((fn () (seven)))'

run macro-expand '(if (= x 0) (print x))' "
  (def list (fn (x . y) (cons x y)))
  (def if-zero (macro (x then) (list 'if (list '= x 0) then)))
  (macro-expand (if-zero x (print x)))"

# sum from 0 to 10
run recursion 55 '(def f (fn (x) (if (= x 0) 0 (+ (f (+ x -1)) x)))) (f 10)'

# string
run string '"asd"' '"asd"'
run string-escape '"a\n\t\"sd"' '"a\n\t\"sd"'

# apply
run apply '3' "(apply + '(1 2))"

# type
run type-int 'int' '(type 1)'
run type-str 'str' '(type "123")'
run type-nil 'nil' '(type nil)'
run type-cons 'cons' '(type (cons 1 2))'
run type-list1 'list' '(type (cons 1 nil))'
run type-list2 'list' '(type (cons 1 (cons 2 nil)))'

# conditionals
run not1 '()' "(not t)"
run not2 '()' "(not 10)"
run not3 't' "(not nil)"
run when1 '2' "(when t 1 2)"
run when2 '()' "(when nil 1 2)"
run unless1 '2' "(unless nil 1 2)"
run unless2 '()' "(unless t 1 2)"

# list fns
run length0 '0' "(length nil)"
run length1 '0' "(length '())"
run length1 '1' "(length '(1))"
run length3 '3' "(length '(1 2 3))"
run reverse0 '()' "(reverse '())"
run reverse1 '(1)' "(reverse '(1))"
run reverse3 '(3 2 1)' "(reverse '(1 2 3))"
run nth0 '1' "(nth '(1 2 3) 0)"
run nth2 '3' "(nth '(1 2 3) 2)"
run empty?1 't' "(empty? nil)"
run empty?2 't' "(empty? '())"
run empty?3 '()' "(empty? '(1))"

# numbers
run range1 '(0)' '(range 0 1)'
run range5 '(0 1 2 3 4)' '(range 0 5)'

# iteration
run dolist1 'ab()' '(dolist (x (list "a" "b")) (print x))'
run dolist2 'ab10' '(dolist (x (list "a" "b") 10) (print x))'
run dotimes1 '0123()' '(dotimes (x 4) (pr x))'
run map '(1 2 3 4 5)' '(map (fn (x) (+ x 1)) (range 0 5))'
run reduce '10' '(reduce + (range 0 5))'
run reduce '15' '(reduce + 5 (range 0 5))'

# alist
run alist?1 "t" "(alist? '())"
run alist?2 "t" "(alist? '((a . 1)))"
run alist?3 "()" "(alist? 'a)"
run alist-has?1 "t" "(alist-has? '((a . 5)) 'a)"
run alist-has?2 "()" "(alist-has? '((a . 5)) 'z)"
run alist-get1 "5" "(alist-get '((a . 5)) 'a)"
run alist-get2 "()" "(alist-get '((a . 5)) 'z)"
run alist-get3 "10" "(alist-get '((a . 5)) 'z 10)"
run alist-set1 "((a . 10))" "(alist-set '((a . 5)) 'a 10)"
run alist-set2 "((a . 10))" "(alist-set '() 'a 10)"
run alist-set3 "((z . 10) (a . 5))" "(alist-set '((a . 5)) 'z 10)"
run alist-del1 "()" "(alist-del '((a . 5)) 'a)"
run alist-del2 "()" "(alist-del '() 'a)"
run alist-del3 "((a . 5))" "(alist-del '((a . 5)) 'z)"

# conditionals (suite)
run or1 't' "(or t nil)"
run or2 '()' "(or nil nil)"
run or3 't' "(or nil nil t nil)"
run or4 '()' "(or)"
run and1 '()' "(and t nil)"
run and2 '()' "(and nil nil)"
run and3 '()' "(and nil nil t nil)"
run and4 't' "(and)"
run and5 't' "(and t)"
run and6 '5' "(and t 1 5)"

# collections
run extend '(1 2 3)' "(extend '(1) '(2 3))"
run extend '(1 2 3 4 5)' "(extend '(1) '() '(2 3 4) '(5))"

# syntax (suite)
run let1 '1' '(let ((x 1)) x)'
