package main

import (
	"flag"
	"log"
	"os"

	"github.com/Neratus/golinq/internal/ast"
	"github.com/Neratus/golinq/internal/codegen"
)

func main() {
	var targetDir string
	var resDir string
	var pkgOverride string

	flag.StringVar(&targetDir, "dir", ".", "корневая директория для поиска запросов")
	flag.StringVar(&resDir, "res", ".", "директория для записи сгенерированных файлов")
	flag.StringVar(&pkgOverride, "pkg", "", "package name for generated file (overrides auto-detection)")
	flag.Parse()

	queries, err := ast.ParseDir(targetDir)
	if err != nil {
		log.Fatalf("failed to parse directory: %v", err)
		os.Exit(-1)
	}
	err = queries.Validate()
	if err != nil {
		log.Fatalf("Validate %v", err)
		os.Exit(-1)
	}
	err = codegen.Generate(queries, resDir, pkgOverride)
	if err != nil {
		log.Fatalf("failed to generate files: %v", err)
		os.Exit(-1)
	}

}
