package main

import (
	"fmt"
	"os"
	"path/filepath"

	"cli-factory/internal/docgen"
)

func main() {
	docsRoot := os.Getenv("CLI_FACTORY_DOCS_ROOT")
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--docs-root" && i+1 < len(os.Args) {
			i++
			docsRoot = os.Args[i]
		}
	}
	if docsRoot == "" {
		docsRoot = filepath.Clean("docs")
	}
	if err := docgen.GenerateTo(docgen.Options{SourceRoot: ".", DocsRoot: docsRoot}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
