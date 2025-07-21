package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/tiborvass/replace"
	"golang.org/x/text/transform"
)

func main() {
	if len(os.Args)%3 != 0 || len(os.Args) < 6 {
		fmt.Fprintf(os.Stderr, "usage: %s <source> <destination> <old_string1> <new_string1> <replace_count1> [...<old_stringN> <new_stringN> <replace_countN>]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "       Reads stream from source and replaces in it replace_count times, old_string with new_string and writes to destination.")
		fmt.Fprintln(os.Stderr, "       If replace_count is -1, it replaces all occurrences.")
		os.Exit(1)
	}
	n := len(os.Args)/3 - 1
	t := make([]transform.Transformer, n)
	for i := range t {
		replaceCount, err := strconv.Atoi(os.Args[5+i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "replace_count must be an integer, received: %q\n", replaceCount)
		}
		t[i] = replace.StringN(os.Args[3+i], os.Args[4+i], replaceCount)
	}
	src, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	dest, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	io.Copy(dest, replace.Chain(src, t...))
}
