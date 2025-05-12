package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	name := flag.String("name", "World", "Name to greet")
	verbose := flag.Bool("verbose", false, "Enable verbose output")

	flag.Parse()

	if err := run(*name, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run(name string, verbose bool) error {
	if verbose {
		fmt.Println("Running in verbose mode")
	}
	fmt.Printf("Hello, %s!\n", name)
	return nil
}
