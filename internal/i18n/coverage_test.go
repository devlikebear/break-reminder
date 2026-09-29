package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Check direct command/help/error output. Protocol serialization, scripts and
// user data use other sinks and must not be translated.
func untranslatedOutput(file *ast.File) []token.Pos {
	var result []token.Pos
	format := regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]|\\[nrt]`)
	prose := regexp.MustCompile(`[A-Za-z가-힣]`)
	check := func(expr ast.Expr) {
		literal, ok := expr.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return
		}
		if value == "break-reminder" || value == "Break Reminder" {
			return
		}
		if prose.MatchString(format.ReplaceAllString(value, "")) {
			result = append(result, literal.Pos())
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.KeyValueExpr:
			if field, ok := n.Key.(*ast.Ident); ok && (field.Name == "Short" || field.Name == "Long") {
				check(n.Value)
			}
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			index := -1
			if pkg.Name == "fmt" {
				switch sel.Sel.Name {
				case "Print", "Println", "Printf", "Errorf":
					index = 0
				case "Fprint", "Fprintln", "Fprintf":
					index = 1
				}
			}
			if pkg.Name == "errors" && sel.Sel.Name == "New" {
				index = 0
			}
			if index >= 0 && index < len(n.Args) {
				check(n.Args[index])
			}
		}
		return true
	})
	return result
}
func TestOutputGuardDetectsUntranslatedProse(t *testing.T) {
	src := `package sample;func sample(){fmt.Println("Timer paused");fmt.Printf("진행: %d", 2);fmt.Println(i18n.Text("Timer"));fmt.Println("%s", name)}`
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(untranslatedOutput(file)); got != 2 {
		t.Fatalf("got %d violations", got)
	}
}
func TestCommandAndErrorOutputIsLocalized(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			// The state writer is a stable KEY=value storage format, not UI output.
			if path == filepath.Join(root, "internal", "state", "state.go") {
				return nil
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, path, nil, 0)
			if err != nil {
				return err
			}
			for _, pos := range untranslatedOutput(file) {
				t.Errorf("%s: wrap display prose in i18n.Text", set.Position(pos))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
