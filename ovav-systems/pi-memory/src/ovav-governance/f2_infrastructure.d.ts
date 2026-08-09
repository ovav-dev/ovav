/**
 * OVAV MEMORY v3 — F2: Infrastructure Gate Validator
 *
 * Verifies that infrastructure files are present:
 * .gitignore, go.mod, VERSION, caps.yaml
 */
export interface ValidationResult {
    valid: boolean;
    errors: string[];
    warnings: string[];
}
export declare function validateInfrastructure(basePath: string): Promise<ValidationResult>;
