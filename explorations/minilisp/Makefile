CC=cc
CFLAGS=-std=c99 -g -O2 -Wall

.PHONY: clean test

shi: shi.c
	$(CC) $(CFLAGS) shi.c -o shi

clean:
	rm -f shi *~

test: shi
	@./test.sh
