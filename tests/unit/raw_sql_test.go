package unit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Every query against the app's database goes through sqlc (internal/db), so a schema change
// breaks `make sqlc` and the build instead of a test at runtime. This reads the AST of every
// non-test .go file and fails on a call that runs a query outside the allowlist below.
//
// Gin's c.Query(key) shares a method name but takes one argument; every query call takes at
// least two (a context and the SQL, or the SQL and its args), which tells them apart.
func TestNoRawSQLOutsideAllowlist(t *testing.T) {
	root := backendRoot(t)

	rawCallNames := map[string]bool{
		"Query": true, "QueryRow": true, "Exec": true, "Prepare": true, "SendBatch": true, "CopyFrom": true,
		"QueryContext": true, "QueryRowContext": true, "ExecContext": true, "PrepareContext": true,
	}

	// internal/db holds the sqlc-generated queries and the transaction helper: the one place
	// hand-written SQL is the point. testkit/database.go builds SQL for any table and columns
	// by design (AssertDatabaseHas).
	allowedDirs := []string{
		filepath.Join(root, "internal", "db"),
	}
	allowedFiles := []string{
		filepath.Join(root, "tests", "testkit", "database.go"),
	}

	var violations []string
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "logs", "uploads", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if isAllowedPath(path, allowedDirs, allowedFiles) {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !rawCallNames[sel.Sel.Name] || len(call.Args) < 2 {
				return true
			}
			pos := fset.Position(call.Pos())
			rel, relErr := filepath.Rel(root, pos.Filename)
			if relErr != nil {
				rel = pos.Filename
			}
			violations = append(violations, fmt.Sprintf("%s:%d: .%s(...)", rel, pos.Line, sel.Sel.Name))
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk backend tree: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("raw SQL outside internal/db and the allowlist (%d); add a query to internal/db/queries instead:\n%s",
			len(violations), strings.Join(violations, "\n"))
	}
}

func isAllowedPath(path string, dirs, files []string) bool {
	for _, f := range files {
		if path == f {
			return true
		}
	}
	for _, dir := range dirs {
		if path == dir || strings.HasPrefix(path, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// backendRoot is backend/ (this file's grandparent) whatever directory the test runs from.
func backendRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
