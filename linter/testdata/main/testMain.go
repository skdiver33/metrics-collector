package main

import (
	"log"
	"os"
)

func main() {
	log.Fatal("It is test") // want

	testPrint := func() {
		testFuc := func() {
			os.Exit(0) // want "call log.Fatal/os.Exit in sub function main function main package"
		}
		testFuc()
		log.Fatal("warning not found") // want "call log.Fatal/os.Exit in sub function main function main package"
	}

	testPrint()
	panic("bybyb") // want "find panic call"
}
