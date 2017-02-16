CC=cc
CV=-std=c11 -D_POSIX_C_SOURCE=201112L
CFLAGS=-g -Os -W -Wall

.PHONY: clean test

shi: src/shi.c deps/*.c deps/libev/ev.o
	$(CC) $(CFLAGS) -o bin/shi src/shi.c deps/linenoise.c deps/pcg_basic.c deps/libev/ev.o

deps/libev/ev.o: deps/libev/*.c deps/libev/*.h
	$(CC) -W -DEV_STANDALONE=1 -o deps/libev/ev.o -c deps/libev/ev.c

clean:
	rm -f bin/shi bin/shi.dSYM deps/libev/ev.o *~

test: shi
	@./test.sh

format:
	clang-format src/shi.c >src/shi.c.new
	mv src/shi.c.new src/shi.c
