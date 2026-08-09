/**
 * OVAV MEMORY v3 — Privacy classifier
 * Preserved from existing Go implementation with minimal changes.
 */
import { PrivacyTag } from "./cell.js";
/**
 * Maps content sensitivity to privacy tags.
 * This is the canonical privacy classification logic.
 */
export function classifyContent(content) {
    const lower = content.toLowerCase();
    const secretPatterns = [
        /api[_-]?key/i,
        /secret/i,
        /password/i,
        /token/i,
        /bearer/i,
        /aws[_-]?access[_-]?key/i,
        /private[_-]?key/i,
        /BEGIN.*PRIVATE.*KEY/i,
        /BEGIN.*RSA.*PRIVATE.*KEY/i
    ];
    for (const pattern of secretPatterns) {
        if (pattern.test(content)) {
            return PrivacyTag.SECRET;
        }
    }
    const sensitivePatterns = [
        /\b\d{3}-\d{2}-\d{4}\b/, // SSN
        /\b\d{16}\b/, // credit card
        /personal/i,
        /health/i,
        /medical/i,
        /salary/i,
        /address/i,
        /phone/i
    ];
    for (const pattern of sensitivePatterns) {
        if (pattern.test(content)) {
            return PrivacyTag.SENSITIVE;
        }
    }
    const projectPatterns = [
        /internal/i,
        /confidential/i,
        /private/i,
        /draft/i,
        /wip/i
    ];
    for (const pattern of projectPatterns) {
        if (pattern.test(content)) {
            return PrivacyTag.PROJECT;
        }
    }
    return PrivacyTag.PUBLIC;
}
/**
 * Determines if a cell can be injected into a given context.
 */
export function canInject(cell, contextScope) {
    if (cell.privacy === PrivacyTag.SECRET) {
        return {
            allowed: false,
            tag: PrivacyTag.SECRET,
            reason: "SECRET cells are never injected"
        };
    }
    if (cell.privacy === PrivacyTag.SENSITIVE) {
        return {
            allowed: false,
            tag: PrivacyTag.SENSITIVE,
            reason: "SENSITIVE cells require explicit context match"
        };
    }
    if (cell.privacy === PrivacyTag.PROJECT && contextScope !== "project") {
        return {
            allowed: false,
            tag: PrivacyTag.PROJECT,
            reason: "PROJECT cells require project-scope context"
        };
    }
    return {
        allowed: true,
        tag: cell.privacy,
        reason: "allowed"
    };
}
/**
 * Filters a list of cells to only those that are injectable in the given scope.
 */
export function filterInjectable(cells, contextScope) {
    return cells.filter(cell => canInject(cell, contextScope).allowed);
}
//# sourceMappingURL=privacy.js.map