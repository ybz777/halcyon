package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	var failed bool
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}
		if !strings.Contains(path, string(filepath.Separator)+"contracts"+string(filepath.Separator)) {
			return nil
		}
		if err := validate(path); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			failed = true
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "walk: %v\n", err)
		os.Exit(1)
	}
	if failed {
		os.Exit(1)
	}
	fmt.Println("OpenAPI validation passed")
}

func validate(path string) error {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	return nil
}
