// Command validate runs the OVAV validator pipeline.
// It executes all registered validators and reports results.
// This is the Go replacement for tools/validators/validate_all.py.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/validators"
)

// findRepoRoot walks up from the current directory to find the OVAV repo root.
// Checks for both .ovav/ and go-runtime/go.mod (both required at real root).
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return findRepoRootFrom(dir)
}

// findRepoRootFrom walks up from the given directory to find the OVAV repo root.
func findRepoRootFrom(startDir string) string {
	dir := startDir
	for range 10 {
		if _, err := os.Stat(filepath.Join(dir, ".ovav")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "go-runtime", "go.mod")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

// filterByID filters validators to only include the one with the specified ID.
// Returns nil if no validator matches.
func filterByID(vals []validators.Validator, id string) []validators.Validator {
	if id == "" {
		return vals
	}
	for _, v := range vals {
		if v.ID() == id {
			return []validators.Validator{v}
		}
	}
	return nil
}

// filterQuick filters validators to only include high-weight ones (weight >= 15).
func filterQuick(vals []validators.Validator) []validators.Validator {
	var quickList []validators.Validator
	for _, v := range vals {
		if v.Weight() >= 15 {
			quickList = append(quickList, v)
		}
	}
	return quickList
}

// formatResultIcon returns the appropriate icon for a validation result status.
func formatResultIcon(status string) string {
	switch status {
	case "fail", "error":
		return "❌"
	case "warn":
		return "⚠️"
	default:
		return "✅"
	}
}

// validationOpts holds the options for runValidation.
type validationOpts struct {
	root         string
	jsonOut      bool
	quick        bool
	id           string
	changedFiles []string
}

// runValidation executes the validator pipeline and writes output to w.
// Returns the count of passed and failed validators, or an error if the
// pipeline cannot start (invalid repo, unknown validator ID).
func runValidation(ctx context.Context, registry *validators.Registry, opts validationOpts, w io.Writer) (passed int, failed int, err error) {
	if !isOVAVRepo(opts.root) {
		return 0, 0, fmt.Errorf("%q is not a valid OVAV repository (requires .ovav/ + go-runtime/go.mod)", opts.root)
	}

	all := registry.All()

	if opts.id != "" {
		filtered := filterByID(all, opts.id)
		if len(filtered) == 0 {
			return 0, 0, fmt.Errorf("validator %q not found", opts.id)
		}
		all = filtered
	}

	if opts.quick {
		all = filterQuick(all)
	}

	results := make([]validators.Result, 0, len(all))

	for _, v := range all {
		result := v.Validate(ctx, opts.root)
		results = append(results, result)

		switch result.Status {
		case "pass":
			passed++
		case "fail", "error":
			failed++
		}

		if opts.jsonOut {
			continue
		}

		icon := formatResultIcon(result.Status)
		fmt.Fprintf(w, "  %s %s: %s\n", icon, result.ID, result.Message)
		for _, issue := range result.Issues {
			fmt.Fprintf(w, "     - %s\n", issue)
		}
	}

	if opts.jsonOut {
		output, _ := json.MarshalIndent(results, "  ", "  ")
		fmt.Fprintln(w, string(output))
	} else {
		fmt.Fprintf(w, "\n  ── Results: %d passed, %d failed ──\n", passed, failed)
	}

	return passed, failed, nil
}

func main() {
	root := flag.String("root", findRepoRoot(), "Repository root directory")
	jsonOut := flag.Bool("json", false, "Output as JSON")
	quick := flag.Bool("quick", false, "Quick mode (skip slow validators)")
	id := flag.String("id", "", "Run only the validator with this ID")
	changedFilesFlag := flag.String("changed-files", "", "Comma-separated list of changed files for scoped validation (validators without scope are skipped)")
	flag.Parse()

	// Parse changed-files comma-separated list
	var changedFiles []string
	if *changedFilesFlag != "" {
		for _, f := range strings.Split(*changedFilesFlag, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				changedFiles = append(changedFiles, f)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	registry := scopeFilteredRegistry(*root, changedFiles)

	_, failed, err := runValidation(ctx, registry, validationOpts{
		root:         *root,
		jsonOut:      *jsonOut,
		quick:        *quick,
		id:           *id,
		changedFiles: changedFiles,
	}, os.Stdout)

	if err != nil {
		fmt.Fprintf(os.Stderr, "validate: %v\n", err)
		os.Exit(1)
	}

	if failed > 0 {
		os.Exit(1)
	}
}

// isOVAVRepo verifies a directory is a valid OVAV repository root.
func isOVAVRepo(dir string) bool {
	_, err1 := os.Stat(filepath.Join(dir, ".ovav"))
	_, err2 := os.Stat(filepath.Join(dir, "go-runtime", "go.mod"))
	return err1 == nil && err2 == nil
}
