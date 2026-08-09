/**
 * OVAV TUI — JSX runtime (development)
 * 
 * Re-exports from jsx-runtime.js for Bun's dev JSX transform.
 */

export { jsx, jsxs, Fragment } from "./jsx-runtime.js";

export function jsxDEV(type, props, key) {
  const { children, ...rest } = props || {};
  if (children !== undefined) {
    return jsx(type, rest, children);
  }
  return jsx(type, rest);
}
