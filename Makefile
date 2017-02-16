CC=cc
CV=-std=c11 -D_POSIX_C_SOURCE=201112L
CFLAGS=-g -Os -W -Wall
DEPS=deps/utf8.c deps/linenoise.c deps/pcg_basic.c deps/libev/ev.o

.PHONY: clean test

shi: src/shi.c deps/*.c deps/libev/ev.o src/prelude.inc
	$(CC) $(CFLAGS) -o bin/shi src/shi.c $(DEPS)

src/prelude.inc: prelude.shi
	rm -f src/prelude.inc
	cat src/prelude.inc.header >>src/prelude.inc
	cat prelude.shi | sed -e 's/\\/\\\\/g;s/"/\\"/g;s/\(.*\)/"\1\\n"/' >>src/prelude.inc
	echo ";\n" >>src/prelude.inc

deps/libev/ev.o: deps/libev/*.c deps/libev/*.h
	$(CC) -W -DEV_STANDALONE=1 -o deps/libev/ev.o -c deps/libev/ev.c

clean:
	rm -f bin/shi bin/shi.dSYM src/prelude.inc deps/libev/ev.o *~

test: shi
	@./test.sh

format:
	clang-format src/shi.c >src/shi.c.new
	mv src/shi.c.new src/shi.c
