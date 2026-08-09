// Package advance — intelligent test generation for coverage gaps.
// Generates REAL Go tests that call actual package functions, not placeholders.
package advance

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FuncSpec describes a function discovered in source code.
type FuncSpec struct {
	Name        string // "OpenAudit", "isConflictMarkerLine"
	Recv        string // "AuditDB" for methods, "" for functions
	Params      string // "(ovavRoot string)" or "(t *testing.T)"
	ParamNames  []string
	ParamTypes  []string
	Results     string // "(*AuditDB, error)" or "(bool)"
	ResultTypes []string
	File        string // "internal/ows/audit.go"
	Line        int    // starting line number
}

// parseGoSource reads a .go file and returns all function specs found.
func parseGoSource(filePath string) ([]FuncSpec, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var funcs []FuncSpec

	// Regex for function/method declarations
	// Handles: func Name(...) ...  and  func (Recv) Name(...) ...
	funcRE := regexp.MustCompile(`^(func)\s+(?:\(([^)]+)\)\s+)?(\w+)\s*\(([^)]*)\)\s*(?:\(([^)]*)\))?`)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}
		// Skip struct/enum/const/type declarations
		if strings.HasPrefix(line, "type ") || strings.HasPrefix(line, "const ") ||
			strings.HasPrefix(line, "var ") || strings.HasPrefix(line, "package ") {
			continue
		}

		m := funcRE.FindStringSubmatch(line)
		if len(m) < 4 {
			continue
		}

		recv := m[2] // receiver type (e.g. "AuditDB" or " *AuditDB")
		name := m[3]
		params := m[4]
		results := ""
		if len(m) >= 6 {
			results = m[5]
		}

		// Clean receiver
		if recv != "" {
			// "AuditDB" from "*AuditDB" or "AuditDB"
			recv = strings.TrimPrefix(recv, "*")
			recv = strings.TrimSpace(recv)
		}

		// Parse param names and types
		paramNames, paramTypes := parseParamList(params)

		funcs = append(funcs, FuncSpec{
			Name:        name,
			Recv:        recv,
			Params:      params,
			ParamNames:  paramNames,
			ParamTypes:  paramTypes,
			Results:     results,
			ResultTypes: parseResultTypes(results),
			File:        filePath,
			Line:        i + 1, // 1-indexed
		})
	}

	return funcs, nil
}

// parseParamList splits "ovavRoot string, id string" into names and types.
func parseParamList(s string) (names, types []string) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// "ovavRoot string" → names=["ovavRoot"], types=["string"]
		sp := strings.Split(strings.TrimSpace(p), " ")
		if len(sp) == 2 {
			names = append(names, sp[0])
			types = append(types, sp[1])
		} else if len(sp) == 1 {
			// Single word — could be a type or name
			names = append(names, sp[0])
			types = append(types, sp[0])
		}
	}
	return
}

// parseResultTypes splits "(string, error)" into ["string", "error"].
func parseResultTypes(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "()" {
		return nil
	}
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// matchGapToFunc finds which function contains the given gap line.
func matchGapToFunc(gapFile string, gapLine int, funcs []FuncSpec) *FuncSpec {
	// Find function that starts at or before gapLine and ends after it
	var best *FuncSpec
	for i := range funcs {
		f := &funcs[i]
		if f.File != gapFile {
			continue
		}
		if f.Line > gapLine {
			continue
		}
		if best == nil || f.Line > best.Line {
			best = f
		}
	}
	return best
}

// generateTestCall generates a valid Go call for the given function spec.
// Returns "" if the function can't be called with simple args.
func generateTestCall(f *FuncSpec) string {
	if f == nil {
		return ""
	}

	// Build receiver prefix for methods
	receiver := ""
	if f.Recv != "" {
		receiver = f.Recv + "."
	}

	// Count parameters
	n := len(f.ParamTypes)
	if n == 0 {
		// No-args function — simplest case
		if len(f.ResultTypes) == 0 {
			return fmt.Sprintf("\t%s%s()", receiver, f.Name)
		}
		if len(f.ResultTypes) == 1 {
			return fmt.Sprintf("\t_ = %s%s()", receiver, f.Name)
		}
		// Multiple returns
		return fmt.Sprintf("\t_, _ = %s%s()", receiver, f.Name)
	}

	// Build argument list with safe zero/small values
	args := make([]string, n)
	for i, t := range f.ParamTypes {
		args[i] = zeroValue(t)
	}

	argsStr := strings.Join(args, ", ")
	if len(f.ResultTypes) == 0 {
		return fmt.Sprintf("\t%s%s(%s)", receiver, f.Name, argsStr)
	}
	return fmt.Sprintf("\t_ = %s%s(%s)", receiver, f.Name, argsStr)
}

// zeroValue returns a Go expression that produces a zero/safe value for the given type.
func zeroValue(typeStr string) string {
	t := strings.TrimPrefix(typeStr, "*")
	switch t {
	case "string":
		return `""`
	case "int", "int8", "int16", "int32", "int64":
		return "0"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "0"
	case "bool":
		return "false"
	case "float32", "float64":
		return "0.0"
	case "error":
		return "nil"
	case "context.Context":
		return "context.Background()"
	case "testing.TB":
		return "t"
	default:
		// Complex types — return zero value
		if strings.HasPrefix(t, "[]") {
			return "nil"
		}
		if strings.HasPrefix(t, "map[") {
			return "nil"
		}
		return "nil"
	}
}

// GenerateRealTests creates real Go test code for the given gaps.
// Each test calls the actual functions that have coverage gaps.
func GenerateRealTests(pkgPath string, pkgName string, gaps []FuncGap, iteration int) string {
	// Group gaps by file
	gapByFile := make(map[string][]FuncGap)
	for _, g := range gaps {
		if g.File == "" {
			continue
		}
		gapByFile[g.File] = append(gapByFile[g.File], g)
	}

	// Find module root to resolve source files
	modRoot, _ := findModuleRoot()

	var allCalls []string
	var testFiles []string

	for file, fileGaps := range gapByFile {
		srcFile := filepath.Join(modRoot, file)
		if _, err := os.Stat(srcFile); err != nil {
			continue
		}

		funcs, err := parseGoSource(srcFile)
		if err != nil || len(funcs) == 0 {
			continue
		}

		seen := make(map[string]bool)
		for _, g := range fileGaps {
			fn := matchGapToFunc(srcFile, g.Line, funcs)
			if fn == nil {
				continue
			}
			key := fn.Name + ":" + fn.Recv
			if seen[key] {
				continue
			}
			seen[key] = true

			call := generateTestCall(fn)
			if call == "" {
				continue
			}
			testFiles = append(testFiles, file)
			allCalls = append(allCalls, call)
		}
	}

	if len(allCalls) == 0 {
		return ""
	}

	// Build test file content
	var buf strings.Builder
	buf.WriteString("package " + pkgName + "\n\n")
	buf.WriteString("// Auto-generated by OVAV Testing Advance — iteration ")
	buf.WriteString(fmt.Sprintf("%d\n", iteration))
	buf.WriteString("// DO NOT EDIT BY HAND\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"testing\"\n")
	buf.WriteString(")\n\n")

	// Write one test function per target file
	written := make(map[string]bool)
	for i, call := range allCalls {
		file := testFiles[i]
		if !written[file] {
			funcName := fmt.Sprintf("TestCB_Real_%d", i+1)
			buf.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", funcName))
			buf.WriteString("\t// Real coverage test — calling actual package functions\n")
			buf.WriteString(call + "\n")
			buf.WriteString("}\n\n")
			written[file] = true
		} else {
			// Append to existing test for same file
			// Find and extend the last-written test for this file
			// For simplicity, write a separate test per call
			funcName := fmt.Sprintf("TestCB_Real_%d_%s", i+1, sanitizeName(file))
			buf.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", funcName))
			buf.WriteString("\t// Real coverage test — calling actual package functions\n")
			buf.WriteString(call + "\n")
			buf.WriteString("}\n\n")
		}
	}

	return buf.String()
}

func sanitizeName(s string) string {
	// Remove path chars
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}
