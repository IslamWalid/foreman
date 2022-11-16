package main

import (
	"fmt"
	"os"

	"github.com/IslamWalid/foreman"
)

func main() {
	foreman, err := foreman.New("./Procfile")
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
