// Package project — isolation.go
//
// Hard isolation gate between OVAV Systems and OVAV Product.
// Prevents sync pipeline cross-contamination: Systems writes to Product
// protected dirs, or Product writes to Systems source directories.
//
// Systems = canonical OVAV source tree (detected by .ovav/ presence)
// Product  = ~/.local/share/ovav/ (installed product, XDG-compliant)
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TargetType identifies whether the sync root is Systems or Product.
type TargetType int

const (
	TargetUnknown TargetType = iota
	TargetSystems
	TargetProduct
)

// String returns a human-readable representation.
func (t TargetType) String() string {
	switch t {
	case TargetSystems:
		return "Systems"
	case TargetProduct:
		return "Product"
	default:
		return "Unknown"
	}
}

// SystemsProtectorates lists directories under .ovav/ that belong exclusively
// to OVAV Systems. They must NEVER be written to from a Product sync root.
// If any of these exist on a Product root, it indicates cross-contamination
// and the operation is hard-blocked.
var SystemsProtectorates = []string{
	".ovav/registry",
	".ovav/plan",
	".ovav/memory",
}

// ValidateTarget detects whether root is Systems or Product, then enforces
// the isolation boundary. Returns an error if the operation would cross
// from one domain into the other.
//
// Systems (.ovav/ present at root): allowed — Systems is the canonical
// source and generates projection artifacts from it.
//
// Product (~/.local/share/ovav/ or XDG_DATA_HOME/ovav/): BLOCKED if
// any Systems protectorates (.ovav/registry/, .ovav/plan/, .ovav/memory/)
// exist on the Product root. Those directories must only live under Systems.
//
// Unknown: allowed to proceed (will fail on missing sources naturally).
func ValidateTarget(root string) error {
	// Resolve symlinks for reliable path comparison.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		resolved, _ = filepath.Abs(root)
	}

	tt := detectTargetType(resolved)

	switch tt {
	case TargetSystems:
		return nil

	case TargetProduct:
		// Product sync must never touch Systems-owned directories.
		if found := findProtectorates(resolved); found != "" {
			return fmt.Errorf(
				"ISOLATION VIOLATION: Product root %q contains Systems-protected directory %q.\n"+
					"Sync on Product must not operate on directories owned by OVAV Systems.\n"+
					"Protected directories: %s\n"+
					"Remove %q from the Product installation and run sync from Systems instead.",
				resolved, found, strings.Join(SystemsProtectorates, ", "), found,
			)
		}
		return nil

	default:
		return nil
	}
}

// detectTargetType classifies root as Systems, Product, or Unknown.
//
// PRODUCT DETECTION RUNS FIRST (priority). A cross-contaminated Product
// directory has a .ovav/ subdirectory from leaked Systems protectorates,
// so checking .ovav/ first would misclassify it as Systems. By checking
// the Product path pattern first, we catch the contamination reliably.
//
// Systems: has .ovav/ directory AND is NOT a Product path.
// Product: matches ~/.local/share/ovav/, $XDG_DATA_HOME/ovav/, or any
//
//	path under XDG_DATA_HOME with basename "ovav".
//
// Unknown: everything else.
func detectTargetType(root string) TargetType {
	// ── Product check (PRIORITY — must run before Systems check) ────
	homeDir, err := os.UserHomeDir()
	if err == nil {
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome == "" {
			xdgDataHome = filepath.Join(homeDir, ".local", "share")
		}

		// Exact match: canonical Product paths
		productDefault := filepath.Join(homeDir, ".local", "share", "ovav")
		productXDG := filepath.Join(xdgDataHome, "ovav")

		productDefaultResolved, _ := filepath.EvalSymlinks(productDefault)
		productXDGResolved, _ := filepath.EvalSymlinks(productXDG)

		if root == productDefaultResolved || root == productXDGResolved {
			return TargetProduct
		}

		// Heuristic: any path under XDG_DATA_HOME with basename "ovav"
		// (catches test directories and non-standard Product layouts).
		xdgResolved, _ := filepath.EvalSymlinks(xdgDataHome)
		if isUnderPath(root, xdgResolved) && filepath.Base(root) == "ovav" {
			return TargetProduct
		}
	}

	// ── Systems check: .ovav/ directory at root level ──────────────────
	// Only reached if Product check did not match. A contaminated Product
	// directory that happens to have .ovav/ will already be caught above.
	if info, err := os.Stat(filepath.Join(root, ".ovav")); err == nil && info.IsDir() {
		return TargetSystems
	}

	return TargetUnknown
}

// findProtectorates checks if any Systems protectorates exist at root.
// Returns the first matched directory path, or empty string if clean.
func findProtectorates(root string) string {
	for _, pd := range SystemsProtectorates {
		target := filepath.Join(root, pd)
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			return pd
		}
	}
	return ""
}

// isUnderPath returns true if child is equal to or under parent.
func isUnderPath(child, parent string) bool {
	if child == parent {
		return true
	}
	// Ensure trailing separator for proper prefix matching.
	if !strings.HasSuffix(parent, string(os.PathSeparator)) {
		parent += string(os.PathSeparator)
	}
	return strings.HasPrefix(child, parent)
}
