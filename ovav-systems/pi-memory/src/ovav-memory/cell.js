/**
 * OVAV MEMORY v3 — Cell type
 * Lightweight event-signature lookup cell for live memory injection.
 */
export const PrivacyTag = {
    PUBLIC: "public",
    PROJECT: "project",
    SENSITIVE: "sensitive",
    SECRET: "secret" // NEVER inject — excluded from all lookups
};
export function isInjectable(cell) {
    return cell.privacy !== PrivacyTag.SECRET && cell.weight > 0;
}
export function createCellId() {
    return crypto.randomUUID();
}
export function buildEventSignature(file, signal, detail) {
    return `${file}:${signal}:${detail}`;
}
//# sourceMappingURL=cell.js.map