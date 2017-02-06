CC=cc
CFLAGS=-std=c99 -ggdb -O2 -Wall

.PHONY: clean test

shi: shi.c
	$(CC) $(CFLAGS) shi.c -o shi

clean:
	rm -f shi *~

test: shi
	@./test.sh

format:
	clang-format shi.c >shi.c.new
	mv shi.c.new shi.c
