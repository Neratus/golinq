package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Neratus/golinq/internal/ast"
	"github.com/Neratus/golinq/internal/codegen"
)

func main() {
	var targetDir string
	var resDir string
	flag.StringVar(&targetDir, "dir", ".", "корневая директория для поиска запросов")
	flag.StringVar(&resDir, "res", ".", "директория для записи сгенерированных файлов")
	flag.Parse()

	queries, err := ast.ParseDir(targetDir)
	if err != nil {
		log.Fatalf("failed to parse directory: %v", err)
	}
	err = queries.Validate()
	if err != nil {
		fmt.Println("Validate ", err)
	}
	err = codegen.Generate(queries, resDir)
	if err != nil {
		log.Fatalf("failed to generate files: %v", err)
	}

}
