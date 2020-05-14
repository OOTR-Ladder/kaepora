package oot_test

import (
	"os"
	"testing"
)

// Keep this in a file without build tags
func TestMain(m *testing.M) {
	if err := os.Chdir("../../.."); err != nil { // generators are CWD dependant
		panic(err)
	}
	os.Exit(m.Run())
}
