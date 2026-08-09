/**
 * OVAV MEMORY v3 — F1: Architecture Gate Validator
 *
 * Verifies that architecture decisions are documented in .ovav/plans/
 * and that key implementation files exist.
 */

import * as fs from "node:fs/promises";
import * as path from "node:path";

export interface ValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

export interface FileCheck {
  path: string;
  pattern?: RegExp;
  required: boolean;
}

const ARCHITECTURE_CHECKS: FileCheck[] = [
  { path: ".ovav/plans/", required: true },
  { path: ".ovav/plan/caps.yaml", required: true },
  { path: "go-runtime/", required: true },
  { path: "go-runtime/go.mod", required: true },
  { path: "go-runtime/internal/", required: true },
  { path: ".ovav/laws/", required: true },
  { path: ".ovav/laws/area_boundary_enforcement.yaml", required: true },
  { path: ".ovav/policy/", required: true },
  { path: ".ovav/policy/permission_authority.json", required: false },
];

const PLAN_FILE_PATTERN = /\.md$/;

export async function validateArchitecture(basePath: string): Promise<ValidationResult> {
  const errors: string[] = [];
  const warnings: string[] = [];

  // Check required paths exist
  for (const check of ARCHITECTURE_CHECKS) {
    const fullPath = path.join(basePath, check.path);
    try {
      const stat = await fs.stat(fullPath);
      if (check.required && !stat) {
        errors.push(`Missing required path: ${check.path}`);
      }
    } catch {
      if (check.required) {
        errors.push(`Missing required path: ${check.path}`);
      } else {
        warnings.push(`Optional path not found: ${check.path}`);
      }
    }
  }

  // Check plans directory has .md files
  const plansPath = path.join(basePath, ".ovav/plans/");
  try {
    const planFiles = await fs.readdir(plansPath);
    const mdFiles = planFiles.filter(f => PLAN_FILE_PATTERN.test(f));
    if (mdFiles.length === 0) {
      errors.push("No plan files (.md) found in .ovav/plans/");
    }
  } catch {
    // Already reported as missing above
  }

  // Check internal/ has .go files
  const internalPath = path.join(basePath, "go-runtime/internal/");
  try {
    const internalFiles = await fs.readdir(internalPath);
    const goFiles = internalFiles.filter(f => f.endsWith(".go"));
    if (goFiles.length === 0) {
      warnings.push("No .go files found in go-runtime/internal/");
    }
  } catch {
    // Already reported as missing above
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}
