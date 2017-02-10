#lang racket/base
(require racket/port
         racket/set
         racket/dict
         racket/function
         "reader.rkt")

(define *shi-version* "0.1.0")

; Reader
; ============================

(current-readtable (make-shi-readtable))

#|
(define-syntax shi-app
  (syntax-rules ()
    [(shi-app f a) (if (hash? f) (hash-ref f a) (f a))]
    [(shi-app f a b) (if (hash? f) (hash-set f a b) (f a b))]
    [(shi-app f . a) (f . a)]))
|#

; Namespace
; ============================

(define *root-namespace* (make-base-empty-namespace))
(define (root-namespace) *root-namespace*)

(define (shi-compile expr env)
  (void))

(define-syntax shi-def
  (syntax-rules ()
    ((shi-def a b)
       (parameterize ((current-namespace (root-namespace)))
         (namespace-set-variable-value! 'a b)))))

(define-syntax shi-defn
  (syntax-rules ()
    ((shi-defn (name . args) body ...)
       (begin
         (shi-def name (lambda args body ...))
         (define (name . args) body ...)))))
;           (apply (parameterize ((current-namespace (root-namespace)))
;                     (namespace-variable-value 'name))
;                  args))))))

; Types
; ============================

(shi-def kwd? keyword?)
(shi-defn (sym? v) (and (symbol? v) (not (keyword? v))))
(shi-defn (nil? v) (void? v))
(shi-defn (bool? v) (boolean? v))
(shi-defn (num? v) (number? v))
(shi-def char? char?)
(shi-defn (str? v) (string? v))
(shi-def list? list?)
(shi-defn (vec? v) (vec? v))
(shi-def hash? dict?)
(shi-defn (fn? v) (procedure? v))
(shi-def port? port?)
(shi-def input-port? input-port?)
(shi-def output-port? output-port?)

(shi-defn (atom? v)
  (or (keyword? v) (void? v) (boolean? v) (number? v) (char? v) (string? v)))

(shi-def identity identity)

(shi-defn (type x)
  (cond
    ((keyword? x) 'kwd)
    ((sym? x) 'sym)
    ((nil? x) 'nil)
    ((bool? x) 'bool)
    ((num? x) 'num)
    ((char? x) 'char)
    ((str? x) 'str)
    ((list? x) 'list)
    ((vec? x) 'vec)
    ((hash? x) 'hash)
    ((fn? x) 'fn)
    ((port? x) 'port)
    (#t (error "type: unknown type" x))))

(define cast-conversions (hash
    'str  (hash
            'str  identity
            'num  number->string
            'char string
            'sym  symbol->string)
    'char (hash
            'char identity
            'num  integer->char)
    'num  (hash
            'num  identity
            'char char->integer)))

(shi-defn (cast x target-type)
  (let* ((x-type (type x))
         (missing-fn (lambda () (error "cast: can't cast" x target-type)))
         (conversions (hash-ref cast-conversions target-type missing-fn))
         (converter (hash-ref conversions x-type missing-fn)))
    (converter x)))

(shi-def eq equal?)

; Symbol
; ============================

(shi-def sym string->symbol)
(shi-def gensym gensym)

; Keyword
; ============================

(shi-def kwd string->keyword)

; Nil
; ============================

(shi-def nil (void))

; Boolean
; ============================

(shi-def true #t)
(shi-def false #f)

; Number
; ============================

; Support adding strings, lists, or numbers together
(shi-def + (lambda args
             (cond
               ((null? args) 0)
               ((or (char? (car args)) (string? (car args)))
                 (apply string-append (map (lambda (x) (cast x 'string)) args)))
               ((list? (car args))
                 (apply append args))
               (#t (apply + args)))))

(shi-def - -)
(shi-def * *)
(shi-def / /)
(shi-def mod modulo)
(shi-def expt expt)
(shi-def sqrt sqrt)
(shi-def gcd gcd)
(shi-def trunc truncate)
(shi-def exact? (lambda (x) (and (integer? x) (exact? x))))

; String
; ============================

(shi-def str (lambda xs
  (string->immutable-string (apply string-append (map (lambda (x) (cast x 'string)) xs)))))

(shi-def str-get string-ref)

(shi-def upper-case string-upcase)
(shi-def lower-case string-downcase)

; List
; ============================

(shi-def list list)

(shi-def length (lambda (x)
  (cond
    ((string? x) (string-length x))
    ((vector? x) (vector-length x))
    ((list? x) (length x))
    (#t (error "length: can't get length of" (type x))))))

; Vector
; ============================

(shi-def vec vector)

; Hash-map
; ============================

(shi-def hash (lambda vs
  (apply hash-map vs)))

; Streams
; ============================

(shi-def eof eof)

(shi-defn (file-port-in fname)
  (open-input-file fname))
(shi-defn (file-port-out fname . args)
  (open-input-file fname 'text (if (equal? args '(append)) 'append 'truncate)))

(shi-def str-port-in open-input-string)
(shi-def str-port-out open-output-string)
(shi-def str-port-output get-output-string)

(shi-def stdout current-output-port)
(shi-def stdin current-input-port)
(shi-def stderr current-error-port)

(shi-defn (with-stdout port fn)
  (parameterize ((current-output-port port)) (fn)))

(shi-defn (with-stdin port fn)
  (parameterize ((current-input-port port)) (fn)))

(shi-defn (read-str)
  (port->string (current-input-port)))

(shi-defn (write-str x)
  (write-string x (current-output-port)))

(shi-defn (close port)
  (cond
    ((input-port? port) (close-input-port port))
    ((output-port? port) (close-output-port port))
    (#t (error "close: given non-port" port))))

; Errors
; ============================

(shi-def ccc call/cc)

(shi-def err error)

(shi-defn (trap-err errfn fn)
  ((call/cc
     (lambda (k)
       (lambda ()
         (with-handlers ([exn:fail? (lambda (ex) (k (lambda () (errfn ex))))])
            (fn)))))))

; Main/REPL
; ============================

(display (namespace-mapped-symbols (root-namespace)))

(define (shi-eval expr)
  (parameterize ((current-namespace (root-namespace))
                 (compile-allow-set!-undefined #t))
    (newline)
    (display (compile expr))
    (newline)
    (eval (compile expr))))

(shi-def eval shi-eval)

(define (run port)
  (let ((expr (read port)))
    (when (not (eof-object? expr))
      (shi-eval expr)
      (run port))))


(define (repl)
  (display "shi> ")
  (let ((expr (read)))
    (when (not (or (eof-object? expr) (eqv? expr ':q)))
      (trap-err
        (lambda (c)
          (parameterize ((current-output-port (current-error-port)))
            ((error-display-handler) (exn-message c) c)))
        (lambda () (write (shi-eval expr))))
      (newline)
      (repl))))

(define (main)
  (if (terminal-port? (current-input-port))
    (begin
      (display (format "Shi v~a REPL\n" *shi-version*))
      (display "Type `:q' or press Ctrl-D to quit.\n")
      (repl)
      (newline))
    (run (current-input-port))))

(main)
