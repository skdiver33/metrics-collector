package testpkg

import (
	"log"
	"os"
)

func main() {
	log.Fatal("It is test") // want "call log.Fatal/os.Exit in non main function of main package."

	testPrint := func() {
		test_Fuc := func() {
			os.Exit(0) // want "call log.Fatal/os.Exit in non main function of main package."
		}
		test_Fuc()
		log.Fatal("warning not found") // want "call log.Fatal/os.Exit in non main function of main package."
	}

	testPrint()
	panic("bybyb") // want "find panic call"
}
