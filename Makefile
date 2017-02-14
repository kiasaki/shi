CC=cc
CV=-std=c11 -D_POSIX_C_SOURCE=201112L
CFLAGS=-g -Os -W -Wall

.PHONY: clean test

shi: shi.c vendor/*.c ev.o
	$(CC) $(CFLAGS) -o shi shi.c vendor/linenoise.c vendor/pcg_basic.c ev.o

ev.o: vendor/libev/*.c vendor/libev/*.h
	$(CC) -W -DEV_STANDALONE=1 -o ev.o -c vendor/libev/ev.c

clean:
	rm -f shi ev.o *~

test: shi
	@./test.sh

format:
	clang-format shi.c >shi.c.new
	mv shi.c.new shi.c
