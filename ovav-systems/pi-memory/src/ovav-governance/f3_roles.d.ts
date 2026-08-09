/**
 * OVAV MEMORY v3 — F3: Roles Gate Validator
 *
 * Verifies that service area roles are defined in caps.yaml
 * and that each area has a lead and scope.
 */
export interface ValidationResult {
    valid: boolean;
    errors: string[];
    warnings: string[];
}
export interface ServiceArea {
    name: string;
    lead: string;
    scope: string[];
}
export declare function validateRoles(basePath: string): Promise<ValidationResult>;
