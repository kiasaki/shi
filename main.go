package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	contents, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	toplevel := Parse("stdin", string(contents))
	fmt.Println(toplevel)
}
