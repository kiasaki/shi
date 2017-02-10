#lang racket/base
(provide make-shi-readtable)
(require racket/port
         racket/set
         syntax/readerr)

(define (make-shi-readtable [rt (current-readtable)])
  (make-readtable rt
                  #\~ #\, #f
                  #\, #\space #f
                  #\_ 'dispatch-macro s-exp-comment-proc
                  #\[ 'terminating-macro vec-proc
                  #\{ 'terminating-macro hash-proc
                  #\\ 'non-terminating-macro char-proc
                  #\: 'non-terminating-macro kw-proc
                  ))

(define (s-exp-comment-proc ch in src ln col pos)
  (make-special-comment (read-syntax/recursive src in)))

(define (vec-proc ch in src ln col pos)
  (define lst-stx
    (parameterize ([read-accept-dot #f])
      (read-syntax/recursive src in ch (make-readtable (current-readtable) ch #\[ #f))))
  (define lst (syntax->list lst-stx))
  (datum->syntax lst-stx (apply vector-immutable lst) lst-stx lst-stx))

(define (hash-proc ch in src ln col pos)
  (define lst-stx
    (parameterize ([read-accept-dot #f])
      (read-syntax/recursive src in ch (make-readtable (current-readtable) ch #\{ #f))))
  (define lst (syntax->list lst-stx))
  (unless (even? (length lst))
    (raise-read-error "hash map literal must contain an even number of forms"
                      src ln col pos (syntax-span lst-stx)))
  (datum->syntax lst-stx (for/hash ([(k v) (in-hash (apply hash lst))])
                           (values (syntax->datum k) v))
    lst-stx
    (syntax-property lst-stx 'shi-hash-map lst-stx)))

(define (char-proc ch in src ln col pos)
  (define in*
    (parameterize ([port-count-lines-enabled #t])
      (input-port-append #f (open-input-string "\\") in)))
  (set-port-next-location! in* ln col pos)
  (read-syntax/recursive src in* #\# #f))

(define (kw-proc ch in src ln col pos)
  (define id-stx
    (read-syntax/recursive src in ch (make-readtable (current-readtable) ch #\: #f)))
  (syntax-property id-stx 'shi-keyword #t))
