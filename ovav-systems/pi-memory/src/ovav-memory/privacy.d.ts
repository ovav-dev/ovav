/**
 * OVAV MEMORY v3 — Privacy classifier
 * Preserved from existing Go implementation with minimal changes.
 */
import { PrivacyTag, type Cell } from "./cell.js";
export interface PrivacyDecision {
    allowed: boolean;
    tag: PrivacyTag;
    reason: string;
}
/**
 * Maps content sensitivity to privacy tags.
 * This is the canonical privacy classification logic.
 */
export declare function classifyContent(content: string): PrivacyTag;
/**
 * Determines if a cell can be injected into a given context.
 */
export declare function canInject(cell: Cell, contextScope: string): PrivacyDecision;
/**
 * Filters a list of cells to only those that are injectable in the given scope.
 */
export declare function filterInjectable(cells: Cell[], contextScope: string): Cell[];
