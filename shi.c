#include <assert.h>
#include <ctype.h>
#include <errno.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <time.h>
#include <unistd.h>

#include <arpa/inet.h>
#include <fcntl.h>
#include <signal.h>
#include <netinet/in.h>
#include <sys/socket.h>

#include "vendor/linenoise.h"
#include "vendor/pcg_basic.h"

#ifndef SIGCHLD
#define SIGCHLD SIGCLD
#endif

static const char *VERSION = "0.1.0";

static __attribute((noreturn)) void error(char *fmt, ...) {
  va_list ap;
  va_start(ap, fmt);
  vfprintf(stderr, fmt, ap);
  fprintf(stderr, "\n");
  va_end(ap);
  exit(1);
}

// {{{ type

// The Lisp object type
enum {
  // Regular objects visible from the user
  TINT = 1,
  TSTR,
  TCELL,
  TSYM,
  TOBJ,
  TPRI,
  TFUN,
  TMAC,
  TENV,

  // Intermediary object only present during GC, points to obj in new semispace.
  TMOVED,

  // Const objects. They are statically allocated and will never be managed by GC.
  TTRUE,
  TNIL,
  TDOT,
  TCPAREN,
};

// Typedef for the primitive function
struct Val;
typedef struct Val *Primitive(void *root, struct Val **env, struct Val **args);

// The object type
typedef struct Val {
  // The first word of the object represents the type of the object. Any code
  // that handles object
  // needs to check its type first, then access the following union members.
  int type;

  // The total size of the object, including "type" field, this field, the
  // contents, and the
  // padding at the end of the object.
  int size;

  // Valect values.
  union {
    // Int
    int intv;
    // String
    char strv[1];
    // Cell
    struct {
      struct Val *car;
      struct Val *cdr;
    };
    // Symbol
    char symv[1];
    // Valect
    // Linked list of association lists containing object properties.
    struct {
      struct Val *props;
      struct Val *proto;
    };
    // Primitive
    Primitive *priv;
    // Function or Macro
    struct {
      struct Val *params;
      struct Val *body;
      struct Val *env;
    };
    // Environment frame
    // Linked list of association lists containing the mapping from symbols to their value.
    struct {
      struct Val *vars;
      struct Val *up;
    };
    // Forwarding pointer
    void *moved;
  };
} Val;

// Globals
FILE *current_file;

// Constants
static Val *True = &(Val){TTRUE, 0, {0}};
static Val *Nil = &(Val){TNIL, 0, {0}};
static Val *Dot = &(Val){TDOT, 0, {0}};
static Val *Cparen = &(Val){TCPAREN, 0, {0}};

// The list containing all symbols. Such data structure is traditionally called
// the "obarray", but I
// avoid using it as a variable name as this is not an array but a list.
static Val *Symbols;

// }}}

// {{{ memory

// The size of the heap in byte
static const unsigned int MEMORY_SIZE = 131072;

// The pointer pointing to the beginning of the current heap
static void *memory;

// The pointer pointing to the beginning of the old heap
static void *from_space;

// The number of bytes allocated from the heap
static size_t mem_nused = 0;

// Flags to debug GC
static bool gc_running = false;
static bool debug_gc = false;
static bool always_gc = false;

static void gc(void *root);

// Currently we are using Cheney's copying GC algorithm, with which the
// available memory is split
// into two halves and all objects are moved from one half to another every time
// GC is invoked. That
// means the address of the object keeps changing. If you take the address of an
// object and keep it
// in a C variable, dereferencing it could cause SEGV because the address
// becomes invalid after GC
// runs.
//
// In order to deal with that, all access from C to Lisp objects will go through
// two levels of
// pointer dereferences. The C local variable is pointing to a pointer on the C
// stack, and the
// pointer is pointing to the Lisp object. GC is aware of the pointers in the
// stack and updates
// their contents with the objects' new addresses when GC happens.
//
// The following is a macro to reserve the area in the C stack for the pointers.
// The contents of
// this area are considered to be GC root.
//
// Be careful not to bypass the two levels of pointer indirections. If you
// create a direct pointer
// to an object, it'll cause a subtle bug. Such code would work in most cases
// but fails with SEGV if
// GC happens during the execution of the code. Any code that allocates memory
// may invoke GC.

#define ROOT_END ((void *)-1)

#define ADD_ROOT(root, size)                                                   \
  void *root_ADD_ROOT_[size + 2];                                              \
  root_ADD_ROOT_[0] = root;                                                    \
  for (int i = 1; i <= size; i++)                                              \
    root_ADD_ROOT_[i] = NULL;                                                  \
  root_ADD_ROOT_[size + 1] = ROOT_END;                                         \
  root = root_ADD_ROOT_

#define DEFINE1(root, var1)                                                    \
  ADD_ROOT(root, 1);                                                           \
  Val **var1 = (Val **)(root_ADD_ROOT_ + 1)

#define DEFINE2(root, var1, var2)                                              \
  ADD_ROOT(root, 2);                                                           \
  Val **var1 = (Val **)(root_ADD_ROOT_ + 1);                                   \
  Val **var2 = (Val **)(root_ADD_ROOT_ + 2)

#define DEFINE3(root, var1, var2, var3)                                        \
  ADD_ROOT(root, 3);                                                           \
  Val **var1 = (Val **)(root_ADD_ROOT_ + 1);                                   \
  Val **var2 = (Val **)(root_ADD_ROOT_ + 2);                                   \
  Val **var3 = (Val **)(root_ADD_ROOT_ + 3)

#define DEFINE4(root, var1, var2, var3, var4)                                  \
  ADD_ROOT(root, 4);                                                           \
  Val **var1 = (Val **)(root_ADD_ROOT_ + 1);                                   \
  Val **var2 = (Val **)(root_ADD_ROOT_ + 2);                                   \
  Val **var3 = (Val **)(root_ADD_ROOT_ + 3);                                   \
  Val **var4 = (Val **)(root_ADD_ROOT_ + 4)

// Round up the given value to a multiple of size. Size must be a power of 2. It
// adds size - 1
// first, then zero-ing the least significant bits to make the result a multiple
// of size. I know
// these bit operations may look a little bit tricky, but it's efficient and
// thus frequently used.
static inline size_t roundup(size_t var, size_t size) {
  return (var + size - 1) & ~(size - 1);
}

// Allocates memory block. This may start GC if we don't have enough memory.
static Val *alloc(void *root, int type, size_t size) {
  // The object must be large enough to contain a pointer for the forwarding
  // pointer. Make it larger if it's smaller than that.
  size = roundup(size, sizeof(void *));

  // Add the size of the type tag and size fields.
  size += offsetof(Val, intv);

  // Round up the object size to the nearest alignment boundary, so that the
  // next object will be allocated at the proper alignment boundary. Currently
  // we align the object at the same boundary as the pointer.
  size = roundup(size, sizeof(void *));

  // If the debug flag is on, allocate a new memory space to force all the
  // existing objects to move to new addresses, to invalidate the old addresses.
  // By doing this the GC behavior becomes more predictable and repeatable. If
  // there's a memory bug that the C variable has a direct reference to a Lisp
  // object, the pointer will become invalid by this GC call. Dereferencing that
  // will immediately cause SEGV.
  if (always_gc && !gc_running)
    gc(root);

  // Otherwise, run GC only when the available memory is not large enough.
  if (!always_gc && MEMORY_SIZE < mem_nused + size)
    gc(root);

  // Terminate the program if we couldn't satisfy the memory request. This can
  // happen if the requested size was too large or the from-space was filled
  // with too many live objects.
  if (MEMORY_SIZE < mem_nused + size)
    error("Memory exhausted");

  // Allocate the object.
  Val *obj = memory + mem_nused;
  obj->type = type;
  obj->size = size;
  mem_nused += size;
  return obj;
}

// }}}

// {{{ gc

// Cheney's algorithm uses two pointers to keep track of GC status. At first
// both pointers point to
// the beginning of the to-space. As GC progresses, they are moved towards the
// end of the
// to-space. The objects before "scan1" are the objects that are fully copied.
// The objects between
// "scan1" and "scan2" have already been copied, but may contain pointers to the
// from-space. "scan2"
// points to the beginning of the free space.
static Val *scan1;
static Val *scan2;

// Moves one object from the from-space to the to-space. Returns the object's
// new address. If the
// object has already been moved, does nothing but just returns the new address.
static inline Val *forward(Val *obj) {
  // If the object's address is not in the from-space, the object is not managed
  // by GC nor it
  // has already been moved to the to-space.
  ptrdiff_t offset = (uint8_t *)obj - (uint8_t *)from_space;
  if (offset < 0 || MEMORY_SIZE <= offset)
    return obj;

  // The pointer is pointing to the from-space, but the object there was a
  // tombstone. Follow the
  // forwarding pointer to find the new location of the object.
  if (obj->type == TMOVED)
    return obj->moved;

  // Otherwise, the object has not been moved yet. Move it.
  Val *newloc = scan2;
  memcpy(newloc, obj, obj->size);
  scan2 = (Val *)((uint8_t *)scan2 + obj->size);

  // Put a tombstone at the location where the object used to occupy, so that
  // the following call
  // of forward() can find the object's new location.
  obj->type = TMOVED;
  obj->moved = newloc;
  return newloc;
}

static void *alloc_semispace() {
  // #include <sys/mman.h>
  // return mmap(NULL, MEMORY_SIZE, PROT_READ | PROT_WRITE, MAP_PRIVATE |
  // MAP_ANON, -1, 0);
  return malloc(MEMORY_SIZE);
}

static char *pr_str(Val *);

// Copies the root objects.
static void forward_root_objects(void *root) {
  Symbols = forward(Symbols);
  for (void **frame = root; frame; frame = *(void ***)frame) {
    for (int i = 1; frame[i] != ROOT_END; i++) {
      if (frame[i]) {
        frame[i] = forward(frame[i]);
      }
    }
  }
}

// Implements Cheney's copying garbage collection algorithm.
// http://en.wikipedia.org/wiki/Cheney%27s_algorithm
static void gc(void *root) {
  assert(!gc_running);
  gc_running = true;

  // Allocate a new semi-space.
  from_space = memory;
  memory = alloc_semispace();

  // Initialize the two pointers for GC. Initially they point to the beginning
  // of the to-space.
  scan1 = scan2 = memory;

  // Copy the GC root objects first. This moves the pointer scan2.
  forward_root_objects(root);

  // Copy the objects referenced by the GC root objects located between scan1
  // and scan2. Once it's
  // finished, all live objects (i.e. objects reachable from the root) will have
  // been copied to
  // the to-space.
  while (scan1 < scan2) {
    switch (scan1->type) {
    case TINT:
    case TSTR:
    case TSYM:
    case TPRI:
      // Any of the above types does not contain a pointer to a GC-managed
      // object.
      break;
    case TCELL:
      scan1->car = forward(scan1->car);
      scan1->cdr = forward(scan1->cdr);
      break;
    case TFUN:
    case TMAC:
      scan1->params = forward(scan1->params);
      scan1->body = forward(scan1->body);
      scan1->env = forward(scan1->env);
      break;
    case TENV:
      scan1->vars = forward(scan1->vars);
      scan1->up = forward(scan1->up);
      break;
    default:
      error("Bug: copy: unknown type %d", scan1->type);
    }
    scan1 = (Val *)((uint8_t *)scan1 + scan1->size);
  }

  // Finish up GC.
  // munmap(from_space, MEMORY_SIZE);
  free(from_space);
  size_t old_nused = mem_nused;
  mem_nused = (size_t)((uint8_t *)scan1 - (uint8_t *)memory);
  if (debug_gc)
    fprintf(stderr, "GC: %zu bytes out of %zu bytes copied.\n", mem_nused,
            old_nused);
  gc_running = false;
}

// }}}

// {{{ constructors

static Val *make_int(void *root, int value) {
  Val *r = alloc(root, TINT, sizeof(int));
  r->intv = value;
  return r;
}

static Val *make_string(void *root, char *value) {
  Val *str = alloc(root, TSTR, strlen(value) + 1);
  strcpy(str->strv, value);
  return str;
}

static Val *cons(void *root, Val **car, Val **cdr) {
  Val *cell = alloc(root, TCELL, sizeof(Val *) * 2);
  cell->car = *car;
  cell->cdr = *cdr;
  return cell;
}

static Val *make_symbol(void *root, char *name) {
  Val *sym = alloc(root, TSYM, strlen(name) + 1);
  strcpy(sym->symv, name);
  return sym;
}

static Val *make_primitive(void *root, Primitive *fn) {
  Val *r = alloc(root, TPRI, sizeof(Primitive *));
  r->priv = fn;
  return r;
}

static Val *make_function(void *root, Val **env, int type, Val **params,
                          Val **body) {
  assert(type == TFUN || type == TMAC);
  Val *r = alloc(root, type, sizeof(Val *) * 3);
  r->params = *params;
  r->body = *body;
  r->env = *env;
  return r;
}

struct Val *make_env(void *root, Val **vars, Val **up) {
  Val *r = alloc(root, TENV, sizeof(Val *) * 2);
  r->vars = *vars;
  r->up = *up;
  return r;
}

// Returns ((x . y) . a)
static Val *acons(void *root, Val **x, Val **y, Val **a) {
  DEFINE1(root, cell);
  *cell = cons(root, x, y);
  return cons(root, cell, a);
}

// }}}

// {{{ util + pretty-print

static int unescape(char *dest, char *src) {
  int i = 0;
  while (*src) {
    switch (*src) {
    case '\n':
      strcat(dest++, "\\n");
      i += 2;
      break;
    case '\r':
      strcat(dest++, "\\r");
      i += 2;
      break;
    case '\t':
      strcat(dest++, "\\t");
      i += 2;
      break;
    case '\"':
      strcat(dest++, "\\\"");
      i += 2;
      break;
    case '\\':
      strcat(dest++, "\\\\");
      i += 2;
      break;
    default:
      *dest = *src;
      i++;
      break;
    }
    src++;
    dest++;
    *dest = '\0';
  }
  return i;
}

static char *pr_str(Val *obj) {
  char *buf = malloc(sizeof(char) * 2048);
  char *s;
  int len = 0;

  switch (obj->type) {
  case TCELL:
    len += sprintf(&buf[len], "(");
    for (;;) {
      s = pr_str(obj->car);
      len += sprintf(&buf[len], "%s", s);
      free(s);
      if (obj->cdr == Nil)
        break;
      if (obj->cdr->type != TCELL) {
        len += sprintf(&buf[len], " . ");
        s = pr_str(obj->cdr);
        len += sprintf(&buf[len], "%s", s);
        free(s);
        break;
      }
      len += sprintf(&buf[len], " ");
      obj = obj->cdr;
    }
    len += sprintf(&buf[len], ")");
    return buf;
  case TSTR:
    len += sprintf(&buf[len], "\"");
    len += unescape(&buf[len], obj->strv);
    len += sprintf(&buf[len], "\"");
    return buf;

#define CASE(type, ...)                                                        \
  case type:                                                                   \
    len += sprintf(&buf[len], __VA_ARGS__);                                    \
    return buf

    CASE(TINT, "%d", obj->intv);
    CASE(TSYM, "%s", obj->symv);
    CASE(TPRI, "<primitive>");
    CASE(TFUN, "<function>");
    CASE(TMAC, "<macro>");
    CASE(TMOVED, "<moved>");
    CASE(TTRUE, "t");
    CASE(TNIL, "()");

#undef CASE

  default:
    // DEBUG
    // len += sprintf(&buf[len], "<tag %d>", obj->type);
    // return buf;
    free(buf);
    error("Bug: print: Unknown tag type: %d", obj->type);
  }
}

// Prints the given object.
static void print(Val *obj) {
  char *str = pr_str(obj);
  printf("%s", str);
  free(str);
}

// Returns the length of the given list. -1 if it's not a proper list.
static int length(Val *list) {
  int len = 0;
  for (; list->type == TCELL; list = list->cdr)
    len++;
  return list == Nil ? len : -1;
}

// Destructively reverses the given list.
static Val *reverse(Val *p) {
  Val *ret = Nil;
  while (p != Nil) {
    Val *head = p;
    p = p->cdr;
    head->cdr = ret;
    ret = head;
  }
  return ret;
}

// May create a new symbol. If there's a symbol with the same name, it will not
// create a new symbol but return the existing one.
static Val *intern(void *root, char *name) {
  for (Val *p = Symbols; p != Nil; p = p->cdr)
    if (strcmp(name, p->car->symv) == 0)
      return p->car;
  DEFINE1(root, sym);
  *sym = make_symbol(root, name);
  Symbols = cons(root, sym, &Symbols);
  return *sym;
}

// }}}

// {{{ reader

#define SYMBOL_MAX_LEN 200
#define STRING_MAX_LEN 1000
const char symbol_chars[] = "~!#$%^&*-_=+:/?<>";

static bool valid_symbol_start_char(char c) {
  return (isalpha(c) || strchr(symbol_chars, c)) && c != '\0';
}

static bool valid_symbol_char(char c) {
  return (isalnum(c) || strchr(symbol_chars, c)) && c != '\0';
}

typedef struct Reader {
  int pos;
  int size;
  char *input;
} Reader;

static Val *reader_expr(Reader *r, void *root);

static Reader *reader_new(char *input) {
  Reader *r = malloc(sizeof(Reader));
  r->pos = -1;
  r->size = strlen(input);
  r->input = malloc(sizeof(char) * (r->size + 1));
  strcpy(r->input, input);
  return r;
}

static void reader_destroy(Reader *r) {
  free(r->input);
  free(r);
}

static int reader_peek(Reader *r) {
  if ((r->pos + 1) == r->size) {
    return EOF;
  }
  return r->input[r->pos + 1];
}

static int reader_next(Reader *r) {
  r->pos++;
  if (r->pos == r->size) {
    return EOF;
  }
  return r->input[r->pos];
}

// Skips the input until newline is found. Newline is one of \r, \r\n or \n.
static void reader_skip_line(Reader *r) {
  for (;;) {
    int c = reader_next(r);
    if (c == EOF || c == '\n') {
      return;
    }
    if (c == '\r') {
      if (reader_peek(r) == '\n') {
        reader_next(r);
      }
      return;
    }
  }
}

// Reads a list. Note that '(' has already been read.
static Val *reader_list(Reader *r, void *root) {
  DEFINE3(root, obj, head, last);
  *head = Nil;
  for (;;) {
    *obj = reader_expr(r, root);
    if (!*obj)
      error("Unclosed parenthesis");
    if (*obj == Cparen)
      return reverse(*head);
    if (*obj == Dot) {
      *last = reader_expr(r, root);
      if (reader_expr(r, root) != Cparen)
        error("Closed parenthesis expected after dot");
      Val *ret = reverse(*head);
      (*head)->cdr = *last;
      return ret;
    }
    *head = cons(root, obj, head);
  }
}

// 'def -> (quote def)
static Val *read_quote(Reader *r, void *root) {
  DEFINE2(root, sym, tmp);
  *sym = intern(root, "quote");
  *tmp = reader_expr(r, root);
  *tmp = cons(root, tmp, &Nil);
  *tmp = cons(root, sym, tmp);
  return *tmp;
}

// `(list a) -> (quasiquote (list a))
static Val *read_quasiquote(Reader *r, void *root) {
  DEFINE2(root, sym, tmp);
  *sym = intern(root, "quasiquote");
  *tmp = reader_expr(r, root);
  *tmp = cons(root, tmp, &Nil);
  *tmp = cons(root, sym, tmp);
  return *tmp;
}

// @b -> (unbox b)
static Val *read_unbox(Reader *r, void *root) {
  DEFINE2(root, sym, tmp);
  *sym = intern(root, "unbox");
  *tmp = reader_expr(r, root);
  *tmp = cons(root, tmp, &Nil);
  *tmp = cons(root, sym, tmp);
  return *tmp;
}

static Val *read_unquote(Reader *r, void *root) {
  DEFINE2(root, sym, tmp);
  if (reader_peek(r) == '@') {
    reader_next(r);
    *sym = intern(root, "unquote-splicing");
  } else {
    *sym = intern(root, "unquote");
  }
  *tmp = reader_expr(r, root);
  *tmp = cons(root, tmp, &Nil);
  *tmp = cons(root, sym, tmp);
  return *tmp;
}

static int read_number(Reader *r, int val) {
  while (isdigit(reader_peek(r))) {
    val = val * 10 + (reader_next(r) - '0');
  }
  return val;
}

static Val *read_string(Reader *r, void *root) {
  char buf[STRING_MAX_LEN + 1];
  int len = 0;
  while (reader_peek(r) != '"' || buf[len - 1] == '\\') {
    if (STRING_MAX_LEN <= len) {
      error("String too long");
    }
    buf[len++] = reader_next(r);

    // handle escapes
    bool is_escape = buf[len - 2] == '\\';
    char current_c = buf[len - 1];
    if (is_escape && current_c == 'n') {
      buf[len - 2] = '\n';
      len--;
    } else if (is_escape && current_c == 'r') {
      buf[len - 2] = '\r';
      len--;
    } else if (is_escape && current_c == 't') {
      buf[len - 2] = '\t';
      len--;
    } else if (is_escape && current_c == '"') {
      buf[len - 2] = '\"';
      len--;
    } else if (is_escape && current_c == '\\') {
      buf[len - 2] = '\\';
      len--;
    }
    // TODO Handle hexadecial char exacpes (\x123)
  }
  buf[len] = '\0';

  // consume closing "
  reader_next(r);

  // create str
  DEFINE1(root, tmp);
  *tmp = make_string(root, buf);
  return *tmp;
}

static Val *read_symbol(Reader *r, void *root, char c) {
  char buf[SYMBOL_MAX_LEN + 1];
  buf[0] = c;
  int len = 1;
  while (valid_symbol_char(reader_peek(r))) {
    if (SYMBOL_MAX_LEN <= len) {
      error("Symbol name too long");
    }
    buf[len++] = reader_next(r);
  }
  buf[len] = '\0';
  return intern(root, buf);
}

static Val *reader_expr(Reader *r, void *root) {
  for (;;) {
    int c = reader_next(r);
    if (c == ' ' || c == '\n' || c == '\r' || c == '\t')
      continue;
    if (c == EOF)
      return NULL;
    if (c == ';') {
      reader_skip_line(r);
      continue;
    }
    if (c == '(')
      return reader_list(r, root);
    if (c == ')')
      return Cparen;
    if (c == '.')
      return Dot;
    if (c == '\'')
      return read_quote(r, root);
    if (c == '`')
      return read_quasiquote(r, root);
    if (c == ',')
      return read_unquote(r, root);
    if (c == '@')
      return read_unbox(r, root);
    if (c == '"')
      return read_string(r, root);
    if (isdigit(c))
      return make_int(root, read_number(r, c - '0'));
    if (c == '-' && isdigit(reader_peek(r)))
      return make_int(root, -read_number(r, 0));
    if (valid_symbol_start_char(c))
      return read_symbol(r, root, c);
    error("Don't know how to handle %c", c);
  }
}

// }}}

// {{{ eval

static Val *eval(void *root, Val **env, Val **obj);

static void add_variable(void *root, Val **env, Val **sym, Val **val) {
  DEFINE2(root, vars, tmp);
  *vars = (*env)->vars;
  *tmp = acons(root, sym, val, vars);
  (*env)->vars = *tmp;
}

// Returns a newly created environment frame.
static Val *push_env(void *root, Val **env, Val **vars, Val **vals) {
  DEFINE3(root, map, sym, val);
  *map = Nil;
  if ((*vars)->type == TSYM) {
    // (fn xs body ...)
    *map = acons(root, vars, vals, map);
  } else {
    // (fn (x y) body ...)
    for (; (*vars)->type == TCELL; *vars = (*vars)->cdr, *vals = (*vals)->cdr) {
      if ((*vals)->type != TCELL)
        error("Cannot apply function: number of argument does not match");
      *sym = (*vars)->car;
      *val = (*vals)->car;
      *map = acons(root, sym, val, map);
    }
    if (*vars != Nil)
      *map = acons(root, vars, vals, map);
  }
  return make_env(root, map, env);
}

// Evaluates the list elements from head and returns the last return value.
static Val *progn(void *root, Val **env, Val **list) {
  DEFINE2(root, lp, r);
  *r = Nil;
  for (*lp = *list; *lp != Nil; *lp = (*lp)->cdr) {
    *r = (*lp)->car;
    *r = eval(root, env, r);
  }
  return *r;
}

// Evaluates all the list elements and returns their return values as a new
// list.
static Val *eval_list(void *root, Val **env, Val **list) {
  DEFINE4(root, head, lp, expr, result);
  *head = Nil;
  for (lp = list; *lp != Nil; *lp = (*lp)->cdr) {
    *expr = (*lp)->car;
    *result = eval(root, env, expr);
    *head = cons(root, result, head);
  }
  return reverse(*head);
}

static bool is_list(Val *obj) { return obj == Nil || obj->type == TCELL; }

static Val *apply_func(void *root, Val **env, Val **fn, Val **args) {
  (void)env;
  DEFINE3(root, params, newenv, body);
  *params = (*fn)->params;
  *newenv = (*fn)->env;
  *newenv = push_env(root, newenv, params, args);
  *body = (*fn)->body;
  return progn(root, newenv, body);
}

// Apply fn with args.
static Val *apply(void *root, Val **env, Val **fn, Val **args, bool do_eval) {
  if (!is_list(*args)) {
    error("apply: argument must be a list");
  }
  if ((*fn)->type == TPRI)
    return (*fn)->priv(root, env, args);
  if ((*fn)->type == TFUN) {
    DEFINE1(root, eargs);
    if (do_eval) {
      *eargs = eval_list(root, env, args);
    } else {
      *eargs = *args;
    }
    return apply_func(root, env, fn, eargs);
  }
  error("apply: not supported");
}

// Searches for a variable by symbol. Returns null if not found.
static Val *find(Val **env, Val *sym) {
  for (Val *p = *env; p != Nil; p = p->up) {
    for (Val *cell = p->vars; cell != Nil; cell = cell->cdr) {
      Val *bind = cell->car;
      if (sym == bind->car)
        return bind;
    }
  }
  return NULL;
}

// Expands the given macro application form.
static Val *macroexpand(void *root, Val **env, Val **obj) {
  if ((*obj)->type != TCELL ||
      ((*obj)->car->type != TSYM && (*obj)->car->type != TMAC)) {
    return *obj;
  }
  DEFINE3(root, bind, macro, args);
  if ((*obj)->car->type == TMAC) {
    *macro = (*obj)->car;
  } else {
    *bind = find(env, (*obj)->car);
    if (!*bind || (*bind)->cdr->type != TMAC)
      return *obj;
    *macro = (*bind)->cdr;
  }
  *args = (*obj)->cdr;
  return apply_func(root, env, macro, args);
}

// Evaluates the S expression.
static Val *eval(void *root, Val **env, Val **obj) {
  switch ((*obj)->type) {
  case TINT:
  case TSTR:
  case TPRI:
  case TFUN:
  case TMAC:
  case TTRUE:
  case TNIL:
    // Self-evaluating objects
    return *obj;
  case TSYM: {
    // Variable
    Val *bind = find(env, *obj);
    if (!bind) {
      error("eval: undefined symbol: %s", (*obj)->symv);
    }
    return bind->cdr;
  }
  case TCELL: {
    // Function application form
    DEFINE3(root, fn, expanded, args);
    *expanded = macroexpand(root, env, obj);
    if (*expanded != *obj)
      return eval(root, env, expanded);
    *fn = (*obj)->car;
    *fn = eval(root, env, fn);
    *args = (*obj)->cdr;
    if ((*fn)->type != TPRI && (*fn)->type != TFUN) {
      error("The head of a list must be a function");
    }
    return apply(root, env, fn, args, true);
  }
  default:
    error("Bug: eval: Unknown tag type: %d", (*obj)->type);
  }
}

// }}}

// {{{ primitives

// {{{ primitives: language

// (do body ...)
static Val *prim_do(void *root, Val **env, Val **list) {
  return progn(root, env, list);
}

// (while cond expr ...)
static Val *prim_while(void *root, Val **env, Val **list) {
  if (length(*list) < 2)
    error("Malformed while");
  DEFINE2(root, cond, exprs);
  *cond = (*list)->car;
  while (eval(root, env, cond) != Nil) {
    *exprs = (*list)->cdr;
    eval_list(root, env, exprs);
  }
  return Nil;
}

static Val *handle_function(void *root, Val **env, Val **list, int type) {
  if ((*list)->type != TCELL ||
      !(is_list((*list)->car) || (*list)->car->type == TSYM) ||
      (*list)->cdr->type != TCELL)
    error("Malformed fn or macro");

  DEFINE2(root, params, body);
  *params = (*list)->car;
  *body = (*list)->cdr;
  Val *p = *params;

  // validate (arg0 arg1) or (arg0 . argN) forms
  if (p->type != TSYM) { // but allow a single symbol to be params
    for (; p->type == TCELL; p = p->cdr)
      if (p->car->type != TSYM)
        error("Parameter must be a symbol");
    if (p != Nil && p->type != TSYM)
      error("Parameter must be a symbol");
  }

  return make_function(root, env, type, params, body);
}

// (fn (<symbol> ...) expr ...)
static Val *prim_fn(void *root, Val **env, Val **list) {
  return handle_function(root, env, list, TFUN);
}

// (macro (<symbol> ...) expr ...)
static Val *prim_macro(void *root, Val **env, Val **list) {
  return handle_function(root, env, list, TMAC);
}

// (def <symbol> expr)
static Val *prim_def(void *root, Val **env, Val **list) {
  if (length(*list) != 2 || (*list)->car->type != TSYM)
    error("Malformed def");
  DEFINE2(root, sym, value);
  *sym = (*list)->car;
  *value = (*list)->cdr->car;
  *value = eval(root, env, value);
  add_variable(root, env, sym, value);
  return *value;
}

// (set <symbol> expr)
static Val *prim_set(void *root, Val **env, Val **list) {
  if (length(*list) != 2 || (*list)->car->type != TSYM)
    error("Malformed set");
  DEFINE2(root, bind, value);
  *bind = find(env, (*list)->car);
  if (!*bind)
    error("Unbound variable %s", (*list)->car->symv);
  *value = (*list)->cdr->car;
  *value = eval(root, env, value);
  (*bind)->cdr = *value;
  return *value;
}

// (pr-str expr)
static Val *prim_pr_str(void *root, Val **env, Val **list) {
  DEFINE2(root, tmp, s);
  *tmp = (*list)->car;
  char *str = pr_str(eval(root, env, tmp));
  *s = make_string(root, str);
  return *s;
}

// (if expr expr expr ...)
static Val *prim_if(void *root, Val **env, Val **list) {
  if (length(*list) < 2)
    error("Malformed if");
  DEFINE3(root, cond, then, els);
  *cond = (*list)->car;
  *cond = eval(root, env, cond);
  if (*cond != Nil) {
    // Test succeded, return then branch and skip evaluatin else
    *then = (*list)->cdr->car;
    return eval(root, env, then);
  }
  *els = (*list)->cdr->cdr;
  if (*els == Nil) {
    // Return nil when else is missing
    return Nil;
  }
  if ((*els)->cdr == Nil) {
    // Return else value if it's last in args (if test then else)
    *then = (*els)->car;
    return eval(root, env, then);
  }
  // Re-enter if with else branch as start (if a ar b br ...)
  return prim_if(root, env, els);
}

// (eq? expr expr)
static Val *prim_eq(void *root, Val **env, Val **list) {
  if (length(*list) != 2)
    error("eq?: needs exactly 2 arguments");
  Val *values = eval_list(root, env, list);
  return values->car == values->cdr->car ? True : Nil;
}

// (eval expr)
static Val *prim_eval(void *root, Val **env, Val **list) {
  if (length(*list) != 1)
    error("Malformed eval");
  DEFINE1(root, arg);
  *arg = (*list)->car;
  //*val = eval(root, env, arg); ??
  return eval(root, env, arg);
}

// (apply fn args)
static Val *prim_apply(void *root, Val **env, Val **list) {
  if (length(*list) != 2)
    error("apply: not given exactly 2 args");
  DEFINE2(root, fn, args);
  *fn = (*list)->car;
  *fn = eval(root, env, fn);

  *args = (*list)->cdr->car;
  *args = eval(root, env, args);
  if ((*args)->type != TCELL)
    error("apply: 2nd argument is not a list");

  return apply(root, env, fn, args, false);
}

// (type expr)
static Val *prim_type(void *root, Val **env, Val **list) {
  if (length(*list) != 1)
    error("Malformed type");
  Val *values = eval_list(root, env, list);

  char *name;

  switch (values->car->type) {
  case TTRUE:
    name = "true";
    break;
  case TNIL:
    name = "nil";
    break;
  case TINT:
    name = "int";
    break;
  case TSTR:
    name = "str";
    break;
  case TCELL:
    if (values->car->cdr != Nil && values->car->cdr->type != TCELL) {
      name = "cons";
    } else {
      name = "list";
    }
    break;
  case TSYM:
    name = "sym";
    break;
  case TPRI:
    name = "prim";
    break;
  case TFUN:
    name = "fn";
    break;
  case TMAC:
    name = "macro";
    break;
  default:
    error("type: unknown object type", values->car->type);
  }

  DEFINE1(root, k);
  *k = intern(root, name);
  return *k;
}

// }}}

// {{{ primitives: marco

// (quote expr)
static Val *prim_quote(void *root, Val **env, Val **list) {
  (void)root;
  (void)env;
  if (length(*list) != 1)
    error("Malformed quote");
  return (*list)->car;
}

// (macro-expand expr)
static Val *prim_macro_expand(void *root, Val **env, Val **list) {
  if (length(*list) != 1)
    error("Malformed macro-expand");
  DEFINE1(root, body);
  *body = (*list)->car;
  return macroexpand(root, env, body);
}

// (gensym)
static Val *prim_gensym(void *root, Val **env, Val **list) {
  (void)env;
  (void)list;
  static int count = 0;
  char buf[16];
  snprintf(buf, sizeof(buf), "G__%d", count++);
  return make_symbol(root, buf);
}

// }}}

// {{{ primitives: list

// (cons expr expr)
static Val *prim_cons(void *root, Val **env, Val **list) {
  if (length(*list) != 2)
    error("Malformed cons");
  Val *cell = eval_list(root, env, list);
  cell->cdr = cell->cdr->car;
  return cell;
}

// (car <cell>)
static Val *prim_car(void *root, Val **env, Val **list) {
  Val *args = eval_list(root, env, list);
  if (args->car->type != TCELL || args->cdr != Nil)
    error("Malformed car: %s", pr_str(args));
  return args->car->car;
}

// (cdr <cell>)
static Val *prim_cdr(void *root, Val **env, Val **list) {
  Val *args = eval_list(root, env, list);
  if (args->car->type != TCELL || args->cdr != Nil)
    error("Malformed cdr");
  return args->car->cdr;
}

// (set-car! <cell> expr)
static Val *prim_set_car(void *root, Val **env, Val **list) {
  DEFINE1(root, args);
  *args = eval_list(root, env, list);
  if (length(*args) != 2 || (*args)->car->type != TCELL)
    error("set_car!: invalid arguments");
  (*args)->car->car = (*args)->cdr->car;
  return (*args)->car;
}

// }}}

// {{{ primitives: string

// (str str0 str1 str3)
static Val *prim_str(void *root, Val **env, Val **list) {
  // Ensure we are only dealing with strings and compute final length
  int len = 0;
  Val *args = eval_list(root, env, list);
  for (Val *a = args; a != Nil; a = a->cdr) {
    if (a->car->type != TSTR)
      error("str: argument not a string");
    len += strlen(a->car->strv);
  }

  char ret[len + 1];
  char *last = &ret[0];

  // Append strings to return value
  for (Val *a = args; a != Nil; a = a->cdr) {
    last = stpcpy(last, a->car->strv);
  }

  ret[len + 1] = '\0';
  return make_string(root, &ret[0]);
}

// (str-len str)
static Val *prim_str_len(void *root, Val **env, Val **list) {
  DEFINE1(root, args);
  *args = eval_list(root, env, list);
  if (length(*args) != 1 || (*args)->car->type != TSTR) {
    error("str-len: 1st arg is not a string");
  }

  return make_int(root, strlen((*args)->car->strv));
}

// }}}

// {{{ primitives: math

// (+ <integer> ...)
static Val *prim_plus(void *root, Val **env, Val **list) {
  int sum = 0;
  for (Val *args = eval_list(root, env, list); args != Nil; args = args->cdr) {
    if (args->car->type != TINT)
      error("+ takes only numbers");
    sum += args->car->intv;
  }
  return make_int(root, sum);
}

// (- <integer> ...)
static Val *prim_minus(void *root, Val **env, Val **list) {
  Val *args = eval_list(root, env, list);
  for (Val *p = args; p != Nil; p = p->cdr)
    if (p->car->type != TINT)
      error("- takes only numbers");
  if (args->cdr == Nil)
    return make_int(root, -args->car->intv);
  int r = args->car->intv;
  for (Val *p = args->cdr; p != Nil; p = p->cdr)
    r -= p->car->intv;
  return make_int(root, r);
}

// (< <integer> <integer>)
static Val *prim_lt(void *root, Val **env, Val **list) {
  Val *args = eval_list(root, env, list);
  if (length(args) != 2)
    error("malformed <");
  Val *x = args->car;
  Val *y = args->cdr->car;
  if (x->type != TINT || y->type != TINT)
    error("< takes only numbers");
  return x->intv < y->intv ? True : Nil;
}

// (= <integer> <integer>)
static Val *prim_num_eq(void *root, Val **env, Val **list) {
  if (length(*list) != 2)
    error("Malformed =");
  Val *values = eval_list(root, env, list);
  Val *x = values->car;
  Val *y = values->cdr->car;
  if (x->type != TINT || y->type != TINT)
    error("= only takes numbers");
  return x->intv == y->intv ? True : Nil;
}

// (rand <integer>)
static Val *prim_rand(void *root, Val **env, Val **list) {
  if (length(*list) != 1)
    error("rand: takes exactly 1 argument");
  Val *values = eval_list(root, env, list);
  Val *x = values->car;
  if (x->type != TINT)
    error("rand: 1st arg is not an int");

  return make_int(root, pcg32_boundedrand(values->car->intv));
}

// }}}

// {{{ primitives: os

// (write "str")
static Val *prim_write(void *root, Val **env, Val **list) {
  if (length(*list) != 2)
    error("write: not given exactly 2 args");

  Val *values = eval_list(root, env, list);

  if (values->car->type != TINT)
    error("write: 1st arg not file descriptor");
  if (values->cdr->car->type != TSTR)
    error("write: 2nd arg not string");

  int fd = values->car->intv;
  char *str = values->cdr->car->strv;

  if (write(fd, str, strlen(str)) < 0)
    error("write: error");
  return Nil;
}

// (read "str")
static Val *prim_read(void *root, Val **env, Val **list) {
  if (length(*list) != 2)
    error("read: not given exactly 2 args");

  Val *values = eval_list(root, env, list);

  if (values->car->type != TINT)
    error("read: 1st arg not file descriptor");
  if (values->cdr->car->type != TINT)
    error("read: 2nd arg not int");

  int fd = values->car->intv;
  int len = values->cdr->car->intv;

  char str[len + 1];
  bzero(str, len + 1);
  if (read(fd, &str, len) < 0)
    error("read: error");

  return make_string(root, str);
}

// (seconds)
static Val *prim_seconds(void *root, Val **env, Val **list) {
  (void)env;
  if (length(*list) != 0)
    error("seconds: takes no args");
  struct timespec spec;
  clock_gettime(CLOCK_REALTIME, &spec);
  return make_int(root, spec.tv_sec);
}

// (sleep n)
static Val *prim_sleep(void *root, Val **env, Val **list) {
  if (length(*list) != 1)
    error("sleep: not given exactly 1 args");
  Val *values = eval_list(root, env, list);
  if (values->car->type != TINT)
    error("sleep: 1st arg not int");

  int milliseconds = values->car->intv;
  struct timespec ts;
  ts.tv_sec = milliseconds / 1000;
  ts.tv_nsec = (milliseconds % 1000) * 1000000;
  nanosleep(&ts, NULL);
  return Nil;
}

// (exit code)
static Val *prim_exit(void *root, Val **env, Val **list) {
  if (length(*list) != 1)
    error("exit: not given exactly 1 args");
  Val *values = eval_list(root, env, list);
  if (values->car->type != TINT)
    error("exit: 1st arg not int");

  exit(values->car->intv);
  return Nil;
}

// (open path append-or-trunc) -> fd
static Val *prim_open(void *root, Val **env, Val **list) {
  if (length(*list) < 1)
    error("open: not given a path");
  Val *values = eval_list(root, env, list);
  if (values->car->type != TSTR)
    error("open: 1st arg not string");


  // Check 2nd param (passed a mode to fopen(3))
  char *mode = "r";
  Val *rest = values->cdr;
  if (rest != Nil && rest->car->type == TSTR) {
    mode = rest->car->strv;
  }

  FILE *fd;
  if ((fd = fopen(values->car->strv, mode)) == NULL) {
    error("open: error opening file");
  }
  return make_int(root, fileno(fd));
}

// (close fd)
static Val *prim_close(void *root, Val **env, Val **list) {
  if (length(*list) != 1)
    error("close: not given exactly 1 arg");
  Val *values = eval_list(root, env, list);
  if (values->car->type != TINT)
    error("open: 1st arg not int");

  if (close(values->car->intv) < 0) {
    error("close: error closing file");
  }
  return Nil;
}

// }}}

// {{{ primitives: net

// (socket domain type protocol) -> fd
static Val *prim_socket(void *root, Val **env, Val **list) {
  if (length(*list) != 3)
    error("socket: not given exactly 3 args");
  Val *values = eval_list(root, env, list);
  if (values->car->type != TINT)
    error("socket: 1st arg not int");
  if (values->cdr->car->type != TINT)
    error("socket: 2nd arg not int");
  if (values->cdr->cdr->car->type != TINT)
    error("socket: 3rd arg not int");

  int domain = values->car->intv;
  int type = values->cdr->car->intv;
  int protocol = values->cdr->cdr->car->intv;

  int fd;
  if ((fd = socket(domain, type, protocol)) < 0) {
    error("socket: error creating socket");
  }

  return make_int(root, fd);
}

// (bind-inet socket-fd host port) -> fd
static Val *prim_bind_inet(void *root, Val **env, Val **list) {
  if (length(*list) != 3)
    error("bind-inet: not given exactly 3 args");
  Val *values = eval_list(root, env, list);
  if (values->car->type != TINT)
    error("bind-inet: 1st arg not int");
  if (values->cdr->car->type != TSTR)
    error("bind-inet: 2nd arg not string");
  if (values->cdr->cdr->car->type != TINT)
    error("bind-inet: 3rd arg not int");

  int socket_fd = values->car->intv;
  char *host = values->cdr->car->strv;
  int port = values->cdr->cdr->car->intv;

  struct sockaddr_in serv_addr;
  serv_addr.sin_family = AF_INET;
  serv_addr.sin_port = htons(port);
  if (inet_aton(host, &serv_addr.sin_addr) < 0) {
    error("bind-inet: could not parse host");
  }
  if (bind(socket_fd, (struct sockaddr *)&serv_addr, sizeof(serv_addr)) < 0) {
    error("bind-inet: error binding to address");
  }

  return Nil;
}

// (listen socket-fd backlog-size)
static Val *prim_listen(void *root, Val **env, Val **list) {
  if (length(*list) != 2)
    error("listen: not given exactly 2 args");
  Val *values = eval_list(root, env, list);
  if (values->car->type != TINT)
    error("listen: 1st arg not int");
  if (values->cdr->car->type != TINT)
    error("listen: 2nd arg not int");

  int socket_fd = values->car->intv;
  int backlog_size = values->cdr->car->intv;

  if (listen(socket_fd, backlog_size) < 0) {
    switch (errno) {
    case EACCES:
      error("listen: insuficient privileges");
    case EBADF:
      error("listen: given socket is not a valid file descriptor");
    case EINVAL:
      error("listen: socket is already listenning");
    case ENOTSOCK:
      error("listen: file descriptor given is not a valid socket");
    case EOPNOTSUPP:
      error("listen: socket type not supported");
    default:
      error("listen: error");
    }
  }

  return Nil;
}

// (accept socket-fd)
static Val *prim_accept(void *root, Val **env, Val **list) {
  if (length(*list) != 1)
    error("accept: not given exactly 1 args");
  Val *values = eval_list(root, env, list);
  if (values->car->type != TINT)
    error("accept: 1st arg not int");

  int client_fd;
  int socket_fd = values->car->intv;
  struct sockaddr_in c_addr;
  socklen_t c_addr_len = sizeof(c_addr);

  if ((client_fd = accept(socket_fd, (struct sockaddr *)&c_addr, &c_addr_len)) <
      0) {
    switch (errno) {
    case EINTR:
      return Nil; // accept interupted by a system call
    case EBADF:
      error("accept: given socket is not a valid file descriptor");
    case EINVAL:
      error("accept: socket is unwilling to accept connections");
    case ENOTSOCK:
      error("accept: file descriptor given is not a valid socket");
    case EOPNOTSUPP:
      error("accept: socket type is not SOCK_STREAM");
    case ENOMEM:
      error("accept: out of memory");
    case EMFILE:
      error("accept: process out of file descriptors");
    case ENFILE:
      error("accept: system out of file descriptors");
    default:
      error("accept: error");
    }
  }

  return make_int(root, client_fd);
}
// }}}

static void add_primitive(void *root, Val **env, char *name, Primitive *fn) {
  DEFINE2(root, sym, prim);
  *sym = intern(root, name);
  *prim = make_primitive(root, fn);
  add_variable(root, env, sym, prim);
}

static void define_constants(void *root, Val **env) {
  ADD_ROOT(root, 8);

  Val **tsym = (Val **)(root_ADD_ROOT_ + 1);
  *tsym = intern(root, "t");
  add_variable(root, env, tsym, &True);

  Val **nsym = (Val **)(root_ADD_ROOT_ + 2);
  *nsym = intern(root, "nil");
  add_variable(root, env, nsym, &Nil);

  Val **system_version = (Val **)(root_ADD_ROOT_ + 3);
  Val **system_version_val = (Val **)(root_ADD_ROOT_ + 4);
  *system_version = intern(root, "*system-version*");
  *system_version_val = make_string(root, (char *)VERSION);
  add_variable(root, env, system_version, system_version_val);

  Val **pf_inet = (Val **)(root_ADD_ROOT_ + 5);
  Val **pf_inet_val = (Val **)(root_ADD_ROOT_ + 6);
  *pf_inet = intern(root, "PF_INET");
  *pf_inet_val = make_int(root, PF_INET);
  add_variable(root, env, pf_inet, pf_inet_val);

  Val **sock_stream = (Val **)(root_ADD_ROOT_ + 7);
  Val **sock_stream_val = (Val **)(root_ADD_ROOT_ + 8);
  *sock_stream = intern(root, "SOCK_STREAM");
  *sock_stream_val = make_int(root, SOCK_STREAM);
  add_variable(root, env, sock_stream, sock_stream_val);
}

static void define_primitives(void *root, Val **env) {
  // Lists
  add_primitive(root, env, "cons", prim_cons);
  add_primitive(root, env, "car", prim_car);
  add_primitive(root, env, "cdr", prim_cdr);
  add_primitive(root, env, "set-car!", prim_set_car);

  // Strings
  add_primitive(root, env, "str", prim_str);
  add_primitive(root, env, "str-len", prim_str_len);

  // Language
  add_primitive(root, env, "def", prim_def);
  add_primitive(root, env, "set", prim_set);
  add_primitive(root, env, "fn", prim_fn);
  add_primitive(root, env, "if", prim_if);
  add_primitive(root, env, "do", prim_do);
  add_primitive(root, env, "while", prim_while);
  add_primitive(root, env, "eq?", prim_eq);
  add_primitive(root, env, "eval", prim_eval);
  add_primitive(root, env, "apply", prim_apply);
  add_primitive(root, env, "type", prim_type);

  // Macro
  add_primitive(root, env, "quote", prim_quote);
  add_primitive(root, env, "gensym", prim_gensym);
  add_primitive(root, env, "macro", prim_macro);
  add_primitive(root, env, "macro-expand", prim_macro_expand);

  // Math
  add_primitive(root, env, "+", prim_plus);
  add_primitive(root, env, "-", prim_minus);
  add_primitive(root, env, "<", prim_lt);
  add_primitive(root, env, "=", prim_num_eq);
  add_primitive(root, env, "rand", prim_rand);

  // OS
  add_primitive(root, env, "pr-str", prim_pr_str);
  add_primitive(root, env, "write", prim_write);
  add_primitive(root, env, "read", prim_read);
  add_primitive(root, env, "seconds", prim_seconds);
  add_primitive(root, env, "sleep", prim_sleep);
  add_primitive(root, env, "exit", prim_exit);
  add_primitive(root, env, "open", prim_open);
  add_primitive(root, env, "close", prim_close);

  // Net
  add_primitive(root, env, "socket", prim_socket);
  add_primitive(root, env, "bind-inet", prim_bind_inet);
  add_primitive(root, env, "listen", prim_listen);
  add_primitive(root, env, "accept", prim_accept);
}
// }}}

// {{{ main

// Returns true if the environment variable is defined and not the empty string.
static bool get_env_flag(char *name) {
  char *val = getenv(name);
  return val && val[0];
}

static char *get_env_value(char *name, char *def) {
  char *val = getenv(name);
  return val && val[0] ? val : def;
}

char *fd_read_all(FILE *fd) {
  int size = 0;
  int chunksize = 1024;
  char *contents = malloc(sizeof(char) * chunksize);

  for (;;) {
    int ret = read(fileno(fd), &contents[size], 128);
    if (ret < 0) {
      perror("fd_read_all");
      exit(1);
    }
    if (ret == 0) {
      contents = realloc(contents, sizeof(char) * (size + 1));
      contents[size] = '\0';
      break;
    }

    size += ret;

    if (size + 128 + 1 >= chunksize) {
      chunksize = chunksize * 2;
      contents = realloc(contents, sizeof(char) * chunksize);
    }
  }
  return contents;
}

char *file_read_all(char *path) {
  FILE *fd = fopen(path, "r");
  if (fd == NULL) {
    error("file_read_all: failed to open file: %s", path);
  }

  // Goto EOF and record file size
  fseek(fd, 0, SEEK_END);
  int size = ftell(fd);
  rewind(fd);

  // Read all bytes
  char *content = (char *)malloc(sizeof(char) * (size + 1));
  int len = fread(content, 1, size, fd);
  if (len < 0) {
    perror("file_read_all");
    exit(1);
  }
  content[len] = '\0';

  fclose(fd);
  return content;
}

Val *eval_reader(Reader *r, void *root, Val **env) {
  DEFINE2(root, val, expr);
  *val = Nil;

  for (;;) {
    *expr = reader_expr(r, root);
    if (!*expr)
      return *val;
    if (*expr == Cparen)
      error("Stray close parenthesis");
    if (*expr == Dot)
      error("Stray dot");
    *val = eval(root, env, expr);
  }
}

Val *eval_input(void *root, Val **env, char *input) {
  DEFINE1(root, val);
  Reader *r = reader_new(input);
  *val = eval_reader(r, root, env);
  reader_destroy(r);

  return *val;
}

char *setup_repl_history() {
  char *hist_folder = get_env_value("HOME", ".");
  char *hist_file = ".shi-history";
  int hist_folder_len = strlen(hist_folder);
  int hist_file_len = strlen(hist_file);

  char *hist_path =
      malloc(sizeof(char) * (hist_folder_len + hist_file_len + 2));
  strcpy(&hist_path[0], hist_folder);
  hist_path[hist_folder_len] = '/';
  strcpy(&hist_path[hist_folder_len + 1], hist_file);
  hist_folder[hist_folder_len + hist_file_len] = '\0';

  linenoiseHistoryLoad(hist_path);

  return hist_path;
}

void setup_repl(void *root, Val **env) {
  DEFINE1(root, val);
  char *line;
  char *hist_path = setup_repl_history();
  linenoiseHistorySetMaxLen(1000);

  while ((line = linenoise("shi> ")) != NULL) {
    if (line[0] != '\0' && line[0] != ',') {
      linenoiseHistoryAdd(line);
      linenoiseHistorySave(hist_path);
      *val = eval_input(root, env, line);
      print(*val);
      printf("\n");
    } else if (!strncmp(line, ",quit", 5)) {
      free(line);
      break;
    } else if (line[0] == ',') {
      printf("Unreconized command: %s\n", line);
    }
    free(line);
  }

  printf("Bye!\n");
  free(hist_path);
}

int main(int argc, char **argv) {
  // Seed rand
  pcg32_srandom(time(NULL) ^ (intptr_t)&printf, (intptr_t)&gc);

  // TODO implement signal primitive
  signal(SIGPIPE, SIG_IGN);
  signal(SIGCHLD, SIG_IGN);

  // Debug flags
  debug_gc = get_env_flag("MINILISP_DEBUG_GC");
  always_gc = get_env_flag("MINILISP_ALWAYS_GC");

  // Memory allocation
  memory = alloc_semispace();

  // Constants and primitives
  Symbols = Nil;
  void *root = NULL;
  DEFINE4(root, env, sh_args_sym, sh_args, sh_arg);
  *env = make_env(root, &Nil, &Nil);
  define_constants(root, env);
  define_primitives(root, env);

  // Register shell args in env
  *sh_args_sym = intern(root, "*args*");
  *sh_args = Nil;
  for (int i = 0; i < argc; i++) {
    *sh_arg = make_string(root, argv[i]);
    *sh_args = cons(root, sh_arg, sh_args);
  }
  *sh_args = reverse(*sh_args);
  add_variable(root, env, sh_args_sym, sh_args);

  // Read and evaluate prelude
  char *prelude_contents = file_read_all("prelude.shi");
  eval_input(root, env, prelude_contents);
  free(prelude_contents);

  // If given a file, read, eval, and exit
  if (argc >= 2) {
    char *file_contents = file_read_all(argv[1]);
    eval_input(root, env, file_contents);
    free(file_contents);
    return 0;
  }

  // If stdin is a file (not terminal) read, and eval
  if (!isatty(fileno(stdin))) {
    char *stdin_contents = fd_read_all(stdin);
    eval_input(root, env, stdin_contents);
    free(stdin_contents);
    return 0;
  }

  // Start REPL
  setup_repl(root, env);
  return 0;
}

// }}}
