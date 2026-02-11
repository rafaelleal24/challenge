package architecture_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const projectImportPath = "github.com/rafaelleal24/challenge"

func TestArchitecturalRules(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatal("Failed to find project root:", err)
	}

	err = filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			fmt.Printf("Failed to parse %s: %v\n", path, err)
			return nil
		}

		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			fmt.Printf("Failed to get relative path for %s: %v\n", path, err)
			return nil
		}
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")

			if isViolation(relPath, importPath) {
				position := fset.Position(imp.Pos())
				fmt.Printf("  VIOLATION FOUND: %s imports %s at %v\n", relPath, importPath, position)
				t.Errorf("ARCHITECTURE VIOLATION at %v: %s imports %s", position, relPath, importPath)
			}
		}

		return nil
	})

	if err != nil {
		t.Fatal("Failed to walk through project files:", err)
	}

}

func isViolation(filePath, importPath string) bool {
	if !strings.Contains(importPath, projectImportPath) {
		return false
	}

	internalImportPath := strings.TrimPrefix(importPath, projectImportPath)
	if !strings.HasPrefix(internalImportPath, "/") {
		internalImportPath = "/" + internalImportPath
	}

	// core/domain can only import third parties libs or golang libs
	if strings.Contains(filePath, "/core/domain") {
		return !strings.Contains(internalImportPath, "/core/domain")
	}

	// core/port can only import domain
	if strings.Contains(filePath, "/core/port") {
		return !strings.Contains(internalImportPath, "/core/domain") && !strings.Contains(internalImportPath, "/core/port")
	}

	//  core/* can only import from inside core
	if strings.Contains(filePath, "/core") &&
		!strings.Contains(filePath, "/core/domain") &&
		!strings.Contains(filePath, "/core/port") {

		return !strings.Contains(internalImportPath, "/core")
	}

	//  inbound adapters cannot import other adapters packages outside of adapters/config

	prefixArr := []string{"/adapters/http"}
	for _, prefix := range prefixArr {
		if strings.Contains(filePath, prefix) {
			if strings.Contains(internalImportPath, "/adapters") {
				return !strings.Contains(internalImportPath, "/adapters/config") &&
					!strings.Contains(internalImportPath, prefix)
			}
		}
	}

	return false
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {

			break
		}
		dir = parent
	}

	currentDir, _ := os.Getwd()
	return currentDir, nil
}
