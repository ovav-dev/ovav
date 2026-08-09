/**
 * OVAV MEMORY v3 — F1: Architecture Gate Validator
 *
 * Verifies that architecture decisions are documented in .ovav/plans/
 * and that key implementation files exist.
 */
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
export declare function validateArchitecture(basePath: string): Promise<ValidationResult>;
