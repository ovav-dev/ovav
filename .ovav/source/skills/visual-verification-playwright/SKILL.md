---
name: visual-verification-playwright
description: "Verify CSS/HTML/React visual bug in running web app with Playwright. Trigger: width shrinks, height jumps, font sizes, layout shifts, container collapses, 'no arreglaste', 'compruebalo', 'muéstrame el ancho'."
license: Apache-2.0
metadata:
  author: dante (OVAV)
  version: "1.0"
---

# Visual Verification with Playwright + Browser Measurement

This skill exists because claiming "should be fixed" on a CSS/HTML visual bug without measured proof leads to user frustration, repeated failed commits, and wasted hours. **The browser's `getBoundingClientRect()` does not lie.**

## When to Use

- User reports a visual bug: width changes, height changes, font size wrong, layout shift
- You applied a CSS fix that targets visual properties (width, height, font, padding, margin, flex, grid)
- Before committing any visual fix (especially after a user rejection cycle)
- When the user's class-name identification is helpful but the actual cause may be in an outer wrapper or a different element

## The Mandate

**After applying any visual CSS fix:**

1. **Measure BEFORE claiming fixed.** Run Playwright in headless mode against the running app with the user's flow (login, navigate, complete the interaction).
2. **Report measured pixel numbers** in a clear table:
   ```
   sl-container:  396 → 310 (-86px) ← still broken
   inner row:     396 → 310 (-86px)
   copySection:   420 → 420 (0px)    ← wrapper OK
   ```
3. **If the diff is non-zero, the fix didn't work.** Iterate. Don't commit a fix that doesn't show 0px diff in the measurement table.

## Login + Measurement Pattern for bt-sys-react (Bitel Agent)

**Credentials** (verified SESSION 57):
- Email: `cpc_williamshs_vtp@bitel.com.pe`
- Password: `Bitel2026` (NOT `Bitel`)
- Vite dev: `http://localhost:5173` with HMR
- Backend Go: `http://localhost:3000` (Vite proxies `/api/*`)

**Playwright path** (always use the project-local one):
```javascript
import { chromium } from '/home/braka/Work/web/products/worktrees/bt-sys-react-design/node_modules/.pnpm/playwright@1.61.1/node_modules/playwright/index.mjs';
```

**Critical flow** (the `#idLlamada` input is `readOnly` — paste is the only way):
```javascript
// 1. Login
await page.fill('#login-email', 'cpc_williamshs_vtp@bitel.com.pe');
await page.fill('#login-password', 'Bitel2026');
await page.click('.login-btn--manual');
await page.waitForSelector('#idLlamada', { timeout: 60000 });

// 2. Paste ID (readonly input)
await page.evaluate(() => {
  const input = document.querySelector('#idLlamada');
  const dt = new DataTransfer();
  dt.setData('text/plain', '12345678');
  input.dispatchEvent(new ClipboardEvent('paste', { clipboardData: dt, bubbles: true }));
});
await page.waitForTimeout(2000);

// 3. Lock the ID
await page.evaluate(() => {
  for (const ic of document.querySelectorAll('.input-inline-icon')) {
    if (!ic.classList.contains('is-locked')) ic.click();
  }
});

// 4. Select INTERNET plantilla
const caso = page.locator('.input-search').first();
await caso.fill('INTERNET');
await page.waitForTimeout(800);
await page.locator('.sd-row').first().click();
await page.waitForTimeout(2000);

// 5. Complete the cascade
const stages = ['dept', 'prov', 'dist'];
for (const stage of stages) {
  await page.locator(`.sl-cascade__cell[data-stage="${stage}"] input`).fill('LIMA');
  await page.waitForTimeout(500);
  await page.locator('.sl-dropdown__item').first().click();
  await page.waitForTimeout(500);
}

// 6. Measure
const widths = await page.evaluate(() => {
  const get = (sel) => {
    const el = document.querySelector(sel);
    if (!el) return null;
    const r = el.getBoundingClientRect();
    const cs = getComputedStyle(el);
    return {
      width: Math.round(r.width * 100) / 100,
      height: Math.round(r.height * 100) / 100,
      minWidth: cs.minWidth,
      flex: cs.flex,
      alignSelf: cs.alignSelf,
      display: cs.display
    };
  };
  return {
    body: get('body'),
    gestorCard: get('.gestor-card'),
    cardBody: get('.card-body'),
    genericSection: get('.generic-section'),
    genericFields: get('.generic-fields'),
    genericField: get('.generic-field'),
    copySection: get('.generic-copy-section--bordered'),
    slContainer: get('.sl-container'),
    cascade: get('.sl-cascade'),
    display: get('.sl-display'),
    segs: Array.from(document.querySelectorAll('.sl-display__seg')).map((s, i) => ({
      idx: i,
      ...get(`[data-seg-idx="${i}"]`) || {
        width: Math.round(s.getBoundingClientRect().width * 100) / 100,
        text: s.textContent
      }
    }))
  };
});
console.log(JSON.stringify(widths, null, 2));
```

## Wrapper-Scope Width Investigation

When inner-element width fix doesn't work, the leak is in an OUTER wrapper. Always walk the FULL chain:

```
body → .gestor-card → .card-body → .generic-section → .generic-fields
  → .generic-copy-section → .sl-container → .sl-display → .sl-display__seg
```

Each level can have `min-width: 0` (flex item default) which allows the chain to shrink at THAT level. The `width: 100% + min-width: 100% + flex: 0 0 100%` lock must be applied at the level that is actually shrinking — often the IMMEDIATE PARENT of the visible symptom, not the deepest target.

**Diagnostic rule**: measure at EVERY level of the chain. The level where width changes between states is where the fix must go.

## Anti-Shrink Pattern (verified SESSION 57)

For elements that must NOT shrink when their content changes:

```css
.element {
  width: 100%;
  min-width: 100%;
  flex: 0 0 100%;         /* grow:0, shrink:0, basis:100% */
  align-self: stretch;     /* force cross-axis stretching */
  box-sizing: border-box;
}
```

`flex: 0 0 100%` is the SHORTHAND that works. Longhand `flex-grow: 1; flex-basis: auto` does NOT (equivalent to `flex: 1 1 auto`, less restrictive).

## Common Mistakes That Cause Silent Failures

- **`!important` everywhere** — masks bugs. Don't use it.
- **`min-width: 0` on children that should preserve width** — kills parent `min-width: 100%` via shrink chain.
- **Longhand `flex-shrink/grow/basis`** — easy to mis-set. Use the `flex` shorthand.
- **Forgetting the immediate parent** — the inner element's `width: 100%` only works if the parent doesn't shrink first.
- **Box-sizing default** — without `box-sizing: border-box`, padding can cause overflow that triggers shrink.

## Output Format

When reporting a visual fix, always include:

```
=== MEASURED PIXEL WIDTHS ===
[before state]
genericField:     420
copySection:      420
slContainer:      396
cascade:          396

[after state]
genericField:     420   (Δ 0.00px)
copySection:      420   (Δ 0.00px)
slContainer:      396   (Δ 0.00px) ← fix verified
cascade:          396   (Δ 0.00px)
```

No fix is complete until the diffs are all 0px for the elements the user identified as the symptom, AND all ancestor elements that were suspected of leaking width.
