package main

import (
	"fmt"

	"github/jungwinter/codeowners"
)

func main() {
	replaced := codeowners.ReplaceAll("* @a\n", "a", "b")
	fmt.Print(replaced)
}
