CC=cc
CFLAGS=-std=c99 -g -O2 -Wall

.PHONY: clean test

shi: shi.c linenoise.c
	$(CC) $(CFLAGS) shi.c linenoise.c -o shi

clean:
	rm -f shi *~

test: shi
	@./test.sh

format:
	clang-format shi.c >shi.c.new
	mv shi.c.new shi.c
