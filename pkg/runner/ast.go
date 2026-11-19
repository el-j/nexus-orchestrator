package runner

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	// goDiagRegex matches standard Go compiler and test errors: "file.go:12:34: message" or "file.go:12: message".
	goDiagRegex = regexp.MustCompile(`(?:^|[\r\n])\s*([a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+):(\d+)(?::(\d+))?:?\s*(.*)`)

	// tsDiagRegex matches TypeScript/ESLint errors: "file.ts(12,34): error TS1234: message".
	tsDiagRegex = regexp.MustCompile(`(?:^|[\r\n])\s*([a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+)\((\d+),(\d+)\):?\s*(.*)`)

	// symbolRegex attempts to extract symbol references like "undefined: Foo" or "in method Bar".
	symbolRegex = regexp.MustCompile(`(?:undefined:\s*|method\s+|function\s+|type\s+|--- FAIL:\s*)([a-zA-Z0-9_]+)`)
)

// ParseDiagnostics extracts structured compiler and test diagnostics from verification output.
func ParseDiagnostics(output string) []Diagnostic {
	var diags []Diagnostic
	seen := make(map[string]bool)

	parseMatch := func(file, lineStr, colStr, msg string) {
		line, _ := strconv.Atoi(lineStr)
		col, _ := strconv.Atoi(colStr)
		cleanFile := filepath.ToSlash(strings.TrimSpace(file))
		cleanMsg := strings.TrimSpace(msg)
		key := fmt.Sprintf("%s:%d:%s", cleanFile, line, cleanMsg)
		if seen[key] || line <= 0 {
			return
		}
		seen[key] = true

		var symbol string
		if symMatch := symbolRegex.FindStringSubmatch(cleanMsg); len(symMatch) > 1 {
			symbol = symMatch[1]
		}

		diags = append(diags, Diagnostic{
			File:    cleanFile,
			Line:    line,
			Column:  col,
			Message: cleanMsg,
			Symbol:  symbol,
		})
	}

	for _, m := range goDiagRegex.FindAllStringSubmatch(output, -1) {
		if len(m) >= 5 {
			parseMatch(m[1], m[2], m[3], m[4])
		}
	}

	for _, m := range tsDiagRegex.FindAllStringSubmatch(output, -1) {
		if len(m) >= 5 {
			parseMatch(m[1], m[2], m[3], m[4])
		}
	}

	return diags
}

// ExtractASTDeclaration parses Go code and returns the declaration enclosing the target line.
func ExtractASTDeclaration(code string, line int) (symbol string, declCode string, ok bool) {
	if strings.TrimSpace(code) == "" || line <= 0 {
		return "", "", false
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, "target.go", code, parser.ParseComments)
	if err != nil {
		return "", "", false
	}

	lines := splitLines(code)

	for _, decl := range fileNode.Decls {
		startPos := fset.Position(decl.Pos())
		endPos := fset.Position(decl.End())

		if line >= startPos.Line && line <= endPos.Line {
			var sym string
			switch d := decl.(type) {
			case *ast.FuncDecl:
				sym = d.Name.Name
				if d.Recv != nil && len(d.Recv.List) > 0 {
					sym = fmt.Sprintf("(%s).%s", recvTypeName(d.Recv.List[0].Type), d.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						sym = ts.Name.Name
						break
					}
				}
			}

			startIdx := startPos.Line - 1
			endIdx := endPos.Line
			declLines := lines[startIdx:endIdx]
			return sym, strings.Join(declLines, "\n"), true
		}
	}

	return "", "", false
}

// ExtractFocusedSnippet extracts lines around target line with line numbers.
func ExtractFocusedSnippet(code string, line, radius int) (snippet string, startLine, endLine int) {
	lines := splitLines(code)
	total := len(lines)
	if total == 0 || line <= 0 {
		return code, 1, total
	}

	start := line - radius
	if start < 1 {
		start = 1
	}
	end := line + radius
	if end > total {
		end = total
	}

	var sb strings.Builder
	for i := start; i <= end; i++ {
		fmt.Fprintf(&sb, "%4d | %s\n", i, lines[i-1])
	}

	return strings.TrimRight(sb.String(), "\n"), start, end
}

// splitLines splits code into individual lines, handling both CRLF and LF.
func splitLines(s string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// recvTypeName returns the string name of a method receiver type.
func recvTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return "*" + id.Name
		}
	}
	return "recv"
}
