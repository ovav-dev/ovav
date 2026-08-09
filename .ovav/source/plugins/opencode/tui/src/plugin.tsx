/**
 * OVAV TUI Plugin v5.0
 * npm package: @ovav/opencode-tui
 * 
 * Correct export format: default export { id, tui }
 * Pattern proven by: opencode-subagent-statusline v0.8.0
 */

import { createSignal, onCleanup } from "solid-js";
import { readFileSync, existsSync } from "node:fs";
import { join } from "node:path";

const PLUGIN_ID = "ovav-statusline.tui";

function readOvavState(dir) {
  if (!dir) return null;
  const path = join(dir, ".ovav", "runtime", "ovav_status.json");
  if (!existsSync(path)) return null;
  try {
    const data = JSON.parse(readFileSync(path, "utf8"));
    return data.ovav || null;
  } catch { return null; }
}

function icon(s) {
  switch (s) {
    case "active": case "pass": return "🟢";
    case "degraded": case "warn": return "🟡";
    case "fail": case "absent": return "🔴";
    default: return "⚫";
  }
}

async function tui(api) {
  const dir = api.state?.path?.directory || "";
  const [s, setS] = createSignal(null);

  const refresh = () => {
    const st = readOvavState(dir);
    if (st) setS(st);
  };

  refresh();
  const iv = setInterval(refresh, 10000);
  onCleanup(() => clearInterval(iv));
  api.lifecycle.onDispose(() => clearInterval(iv));
  api.event.on("session.created", refresh);
  api.event.on("session.status", refresh);

  api.slots.register({
    order: 100,
    slots: {
      session_prompt_right(_ctx) {
        const st = s();
        if (!st || st.overall === "absent") return null;
        return `${icon(st.governor?.status)} OVAV  ${icon(st.memory?.status)} MEM  ${st.branch?.branch}`;
      },
      sidebar_content(_ctx) {
        const st = s();
        if (!st) return null;
        return [
          `🛡️ Governor: ${st.governor?.icon} ${st.governor?.label}`,
          `🧠 Memory:   ${st.memory?.icon} ${st.memory?.label}`,
          `📊 Branch:   ${st.branch?.branch}`,
          `   Tokens:   ${st.tokens?.total_all?.toLocaleString?.() || "—"}`,
        ].join("\n");
      },
      home_bottom(_ctx) {
        const st = s();
        if (!st) return null;
        return `${st.icon} OVAV ${st.overall} | ${st.branch?.branch}`;
      },
    },
  });
}

export default { id: PLUGIN_ID, tui };
