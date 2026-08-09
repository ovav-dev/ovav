/**
 * OVAV MEMORY v3 — F2: Infrastructure Gate Validator
 *
 * Verifies that infrastructure files are present:
 * .gitignore, go.mod, VERSION, caps.yaml
 */

import * as fs from "node:fs/promises";
import * as path from "node:path";

export interface ValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

const INFRA_CHECKS = [
  ".gitignore",
  "go-runtime/go.mod",
  ".ovav/plan/caps.yaml",
  "go-runtime/cmd/",
];

const INFRA_FILES = [
  "VERSION",
  "CHANGELOG.md",
];

export async function validateInfrastructure(basePath: string): Promise<ValidationResult> {
  const errors: string[] = [];
  const warnings: string[] = [];

  for (const check of INFRA_CHECKS) {
    const fullPath = path.join(basePath, check);
    try {
      await fs.access(fullPath);
    } catch {
      errors.push(`Missing infrastructure file: ${check}`);
    }
  }

  for (const file of INFRA_FILES) {
    const fullPath = path.join(basePath, file);
    try {
      await fs.access(fullPath);
    } catch {
      warnings.push(`Infrastructure file not found: ${file} (optional)`);
    }
  }

  // Validate go.mod has a valid module name
  const goModPath = path.join(basePath, "go-runtime/go.mod");
  try {
    const content = await fs.readFile(goModPath, "utf-8");
    if (!content.startsWith("module ")) {
      errors.push("go.mod does not start with 'module' declaration");
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
