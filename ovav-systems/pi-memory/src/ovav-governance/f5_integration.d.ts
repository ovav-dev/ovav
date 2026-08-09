/**
 * OVAV MEMORY v3 — F5: Integration Gate Validator
 *
 * Runs all F0-F4 validators in sequence and computes
 * a combined integrity score.
 */
import { type ValidationResult as F1Result } from "./f1_architecture.js";
import { type ValidationResult as F2Result } from "./f2_infrastructure.js";
import { type ValidationResult as F3Result } from "./f3_roles.js";
import { type ValidationResult as F4Result } from "./f4_security.js";
export interface ValidationResult {
    valid: boolean;
    errors: string[];
    warnings: string[];
}
export interface F5Report {
    score: number;
    status: "HEALTHY" | "DEGRADED" | "CRITICAL";
    f1: F1Result;
    f2: F2Result;
    f3: F3Result;
    f4: F4Result;
    totalErrors: number;
    totalWarnings: number;
}
export declare function validateF5(basePath: string): Promise<F5Report>;
export declare function validateF0(_basePath: string): Promise<ValidationResult>;
