package main

import (
	"os"
	"testing"
)

func TestMainRunsWithHelp(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"taskservice", "--help"}
	defer func() { os.Args = origArgs }()

	main()
}
