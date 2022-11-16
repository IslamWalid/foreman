package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/IslamWalid/foreman"
)

func main() {
	verbosePtr := flag.Bool("v", false, "run the program verbosely")
	procfilePtr := flag.String("f", "Procfile", "specify the procfile path")
	flag.Parse()

	foreman, err := foreman.New(*procfilePtr, *verbosePtr)
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
