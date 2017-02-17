# shi lisp

## TODO

### Big projects

- Async I/O
- Async Net/Socket
- Module System

### Other

- prim: str-part
- prim: read-sexp
- http static files
- http router
- http responses
- json
- regexp

### Stdlib

**Environment**

- def-global
- globals

**Symbol**

- sym

**String**

- str
- str-len
- str-eq?
- str-join
- str-split
- str-sub
- str-upcase
- str-downcase
- str->number
- number->string
- str->bytes
- bytes->str

**Time**

- Time:init(y, m, d, h, m, s, sss)
- Time:now()
- Time:parse(f, str)
- time:format(f)
- time:after(t)
- time:before(t)
- time:add(amt, unit)
- time:sub(amt, unit)

**IO**

- io/open
- io/close
- io/flush
- io/read
- io/read-bytes
- io/write
- io/write-bytes
- io/read-all

**OS**

- *args*
- os/exit
- os/socket
- os/bind-inet
- os/accept
- os/set-blocking
- os/getenv
- os/tty?

**Path**

- path/exists?
- path/dir?
- path/abs
- path/cwd
- path/join -> str-join patsep str?
- path/split -> str-split patsep str?

**JSON**

- json/parse(str)
- json/format(str)

**Random**

- random/n
