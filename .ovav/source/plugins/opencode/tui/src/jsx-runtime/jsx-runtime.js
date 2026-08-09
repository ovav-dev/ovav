/**
 * OVAV TUI — JSX runtime (production)
 * 
 * Provides jsx/jsxs functions for @opentui/solid JSX type system.
 * Uses solid-js/h internally.
 */

import h from "solid-js/h";

// Fragment: renders children without a wrapper element
export const Fragment = (props) => props?.children ?? null;

export function jsx(type, props, key) {
  const { children, ...rest } = props || {};
  if (children !== undefined) {
    return h(type, rest, children);
  }
  return h(type, rest);
}

export function jsxs(type, props, key) {
  return jsx(type, props, key);
}
