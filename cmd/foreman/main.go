package main

import (
	"fmt"
	"os"

	"github.com/IslamWalid/foreman"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "too few arguments: specify the procile path")
		os.Exit(1)
	}

	foreman, err := foreman.New(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = foreman.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
