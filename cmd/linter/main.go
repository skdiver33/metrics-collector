package main

import (
	linter "github.com/skdiver33/metrics-collector/linter"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(linter.MyAnalyzer) }
