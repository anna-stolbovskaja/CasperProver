package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "prove":
		fmt.Println("prove: not yet implemented")
	case "verify":
		fmt.Println("verify: not yet implemented")
	case "revoke":
		fmt.Println("revoke: not yet implemented")
	case "demo":
		fmt.Println("demo: not yet implemented")
	case "agent":
		fmt.Println("agent: not yet implemented")
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: casperprover <command>\n\nCommands:\n  prove   Generate verifiable proof\n  verify  Verify existing proof\n  revoke  Revoke a proof\n  demo    Run demo scenario\n  agent   Manage agent registration\n")
}
