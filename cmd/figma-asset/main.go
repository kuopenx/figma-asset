package main

import (
	"fmt"
	"os"

	"github.com/kuopenx/figma-asset/internal/figmaasset"
)

var Version = "dev"

func main() {
	if err := figmaasset.Run(os.Args[1:], Version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
