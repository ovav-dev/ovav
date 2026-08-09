/**
 * OVAV Premium Input — TUI COMPONENTS v2.0 + HOT-RELOAD
 * 
 * MEJORAS v2.0:
 * - Command Palette con SelectList (componente TUI real)
 * - Shortcuts Overlay con Container
 * - Status Bar theme-aware
 * - Working Indicator personalizado
 * - Footer con git info rica
 * - AUTO-RELOAD: Detecta cambios y recarga automáticamente
 */

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { 
  Container, 
  Text, 
  Spacer, 
  SelectList, 
  DynamicBorder,
  type SelectItem,
  matchesKey, 
  Key,
  truncateToWidth 
} from "@earendil-works/pi-tui";
import { Type } from "typebox";
import * as fs from "node:fs";
import * as path from "node:path";
import { execSync } from "node:child_process";

export default function (pi: ExtensionAPI) {
  // ═══════════════════════════════════════════════════════════════════════════
  // STATE
  // ═══════════════════════════════════════════════════════════════════════════
  
  let projectInfo: ProjectInfo = {
    name: "unknown",
    path: "unknown",
    branch: "unknown",
    isWorktree: false,
    worktreeRoot: "unknown",
    hasChanges: false,
  };

  interface ProjectInfo {
    name: string;
    path: string;
    branch: string;
    isWorktree: boolean;
    worktreeRoot: string;
    hasChanges: boolean;
  }

  // Hot-reload state
  let hotReloadWatcher: fs.FSWatcher | null = null;
  let hotReloadDebounceTimer: NodeJS.Timeout | null = null;
  const HOT_RELOAD_DELAY = 500; // ms

  // ═══════════════════════════════════════════════════════════════════════════
  // HOT-RELOAD SYSTEM
  // ═══════════════════════════════════════════════════════════════════════════
  
  function startHotReloadWatcher(ctx: any, extensionPath: string) {
    // Stop existing watcher if any
    stopHotReloadWatcher();
    
    // Track file modification time to avoid self-triggering
    let lastTriggerTime = Date.now();
    
    console.log(`[OVAV Premium] Starting hot-reload watcher on: ${extensionPath}`);
    
    try {
      hotReloadWatcher = fs.watch(extensionPath, { persistent: false }, (eventType, filename) => {
        if (eventType !== "change") return;
        if (!filename || !filename.endsWith(".ts")) return;
        
        // Debounce to avoid rapid-fire reloads
        if (hotReloadDebounceTimer) {
          clearTimeout(hotReloadDebounceTimer);
        }
        
        hotReloadDebounceTimer = setTimeout(async () => {
          const now = Date.now();
          // Minimum 2 seconds between reloads to avoid loops
          if (now - lastTriggerTime < 2000) {
            console.log(`[OVAV Premium] Skipping reload (too soon)`);
            return;
          }
          lastTriggerTime = now;
          
          console.log(`[OVAV Premium] Detected change in: ${filename}`);
          
          // Show prominent reload notification
          ctx.ui.setWidget("ovav-reload", [
            "",
            "╔══════════════════════════════════════════════════════════════════╗",
            "║                                                                  ║",
            "║       🔄 OVAV PREMIUM INPUT — HOT-RELOAD DETECTADO 🔄          ║",
            "║                                                                  ║",
            "║              Recargando extensión automáticamente...               ║",
            "║                                                                  ║",
            "╚══════════════════════════════════════════════════════════════════╝",
            "",
          ]);
          ctx.ui.notify(`🔥 Reload: ${filename}`, "info");
          
          try {
            await ctx.reload();
            // Clear reload widget
            ctx.ui.setWidget("ovav-reload", []);
            ctx.ui.notify(`✅ Extensión recargada!`, "success");
            ctx.ui.notify(`🔥 Cambios aplicados - mira la nueva UI`, "success");
            console.log(`[OVAV Premium] Hot-reload successful`);
          } catch (err) {
            ctx.ui.notify(`❌ Hot-reload failed: ${err}`, "error");
            console.error(`[OVAV Premium] Hot-reload error:`, err);
          }
        }, HOT_RELOAD_DELAY);
      });
      
      hotReloadWatcher.on("error", (err) => {
        console.error(`[OVAV Premium] Watcher error:`, err);
      });
      
    } catch (err) {
      console.error(`[OVAV Premium] Failed to start watcher:`, err);
    }
  }
  
  function stopHotReloadWatcher() {
    if (hotReloadDebounceTimer) {
      clearTimeout(hotReloadDebounceTimer);
      hotReloadDebounceTimer = null;
    }
    if (hotReloadWatcher) {
      hotReloadWatcher.close();
      hotReloadWatcher = null;
    }
  }
  
  // ═══════════════════════════════════════════════════════════════════════════
  // COMMANDS REGISTRY
  // ═══════════════════════════════════════════════════════════════════════════
  
  const commands: Array<{ cmd: string; desc: string; cat: string }> = [
    { cmd: "/ovav daily", desc: "Estado del sistema", cat: "OVAV" },
    { cmd: "/ovav next", desc: "Siguiente tarea", cat: "OVAV" },
    { cmd: "/ovav validate", desc: "Validar workspace", cat: "OVAV" },
    { cmd: "/ovav memory", desc: "Buscar memorias", cat: "OVAV" },
    { cmd: "/ovav check", desc: "Verificar integridad", cat: "OVAV" },
    { cmd: "/git status", desc: "Estado git", cat: "Git" },
    { cmd: "/git diff", desc: "Cambios pendientes", cat: "Git" },
    { cmd: "/git log", desc: "Historial", cat: "Git" },
    { cmd: "/git commit", desc: "Crear commit", cat: "Git" },
    { cmd: "/review", desc: "Code review", cat: "PIAGENT" },
    { cmd: "/component", desc: "Crear componente", cat: "PIAGENT" },
    { cmd: "/test", desc: "Generar tests", cat: "PIAGENT" },
    { cmd: "/compact", desc: "Compactar contexto", cat: "PIAGENT" },
    { cmd: "/deploy", desc: "Deploy", cat: "PIAGENT" },
  ];

  // Convert to SelectItem format for SelectList
  function getCommandItems(): SelectItem[] {
    return commands.map(c => ({
      value: c.cmd,
      label: c.cmd,
      description: `[${c.cat.padEnd(8)}] ${c.desc}`,
    }));
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // GIT + WORKTREE DETECTION
  // ═══════════════════════════════════════════════════════════════════════════
  
  function detectGitInfo(cwd: string): ProjectInfo {
    const info: ProjectInfo = {
      name: path.basename(cwd),
      path: cwd,
      branch: "unknown",
      isWorktree: false,
      worktreeRoot: cwd,
      hasChanges: false,
    };

    const gitFile = path.join(cwd, ".git");
    
    let isWorktree = false;
    let gitRoot = cwd;
    
    if (fs.existsSync(gitFile)) {
      try {
        const content = fs.readFileSync(gitFile, "utf8").trim();
        if (content.startsWith("gitdir:")) {
          isWorktree = true;
          const gitdirPath = content.replace("gitdir: ", "").replace("gitdir: ", "");
          const worktreesMatch = gitdirPath.match(/\.git\/worktrees\/(.+)$/);
          if (worktreesMatch) {
            const worktreeConfig = path.join(path.dirname(gitdirPath), "config");
            if (fs.existsSync(worktreeConfig)) {
              const config = fs.readFileSync(worktreeConfig, "utf8");
              const lines = config.split("\n");
              for (let i = 0; i < lines.length; i++) {
                if (lines[i].includes("worktree")) {
                  const nextLines = lines.slice(i, i + 3).join("\n");
                  const pathMatch = nextLines.match(/path\s*=\s*(.+)/);
                  if (pathMatch) {
                    info.path = pathMatch[1].trim();
                    info.name = path.basename(info.path);
                    break;
                  }
                }
              }
            }
          }
          gitRoot = path.dirname(path.dirname(gitdirPath));
        }
      } catch {}
    }

    info.isWorktree = isWorktree;
    info.worktreeRoot = gitRoot;

    try {
      const branch = execSync("git rev-parse --abbrev-ref HEAD 2>/dev/null", { 
        cwd, encoding: "utf8", timeout: 2000 
      }).trim();
      info.branch = branch;
    } catch {}

    try {
      const status = execSync("git status --porcelain 2>/dev/null", { 
        cwd, encoding: "utf8", timeout: 2000 
      });
      info.hasChanges = status.trim().length > 0;
    } catch {}

    const owcWorktrees = path.join(gitRoot, ".ovav", "worktrees");
    if (fs.existsSync(owcWorktrees)) {
      try {
        const worktrees = fs.readdirSync(owcWorktrees);
        for (const wt of worktrees) {
          const wtPath = path.join(owcWorktrees, wt);
          if (fs.statSync(wtPath).isDirectory()) {
            if (cwd.startsWith(wtPath) || cwd === wtPath) {
              info.isWorktree = true;
              info.path = wtPath;
              info.name = wt;
              info.worktreeRoot = gitRoot;
              break;
            }
          }
        }
      } catch {}
    }

    return info;
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // TUI COMPONENT: COMMAND PALETTE
  // ═══════════════════════════════════════════════════════════════════════════
  
  class CommandPalette {
    private container: Container;
    private selectList: SelectList;
    private items: SelectItem[];
    private onSelect: ((cmd: string) => void) | undefined;
    private onCancel: (() => void) | undefined;
    private cachedWidth?: number;
    private cachedLines?: string[];
    private theme: any;
    
    constructor(theme: any, items: SelectItem[], onSelect?: (cmd: string) => void, onCancel?: () => void) {
      this.theme = theme;
      this.items = items;
      this.onSelect = onSelect;
      this.onCancel = onCancel;
      
      this.container = new Container();
      
      // Title with accent border
      const titleText = theme.fg("accent", "╔══ COMMAND PALETTE ══╗");
      this.container.addChild(new Text(titleText, 0, 0));
      
      // SelectList with theme-aware styling
      this.selectList = new SelectList(items, Math.min(items.length, 12), {
        selectedPrefix: (t) => theme.fg("accent", t),
        selectedText: (t) => theme.fg("text", t),
        description: (t) => theme.fg("muted", t),
        scrollInfo: (t) => theme.fg("dim", t),
        noMatch: (t) => theme.fg("warning", t),
      });
      this.selectList.onSelect = (item) => {
        this.onSelect?.(item.value);
      };
      this.selectList.onCancel = () => {
        this.onCancel?.();
      };
      this.container.addChild(this.selectList);
      
      // Help text
      const helpText = theme.fg("dim", "↑↓ navigate  ↵ select  Tab=search  Esc=cancel");
      this.container.addChild(new Text(helpText, 0, 0));
      
      // Bottom border
      const bottomText = theme.fg("accent", "╚═══════════════════════════════╝");
      this.container.addChild(new Text(bottomText, 0, 0));
    }
    
    handleInput(data: string): void {
      if (matchesKey(data, Key.escape)) {
        this.onCancel?.();
        return;
      }
      this.selectList.handleInput(data);
    }
    
    render(width: number): string[] {
      if (this.cachedLines && this.cachedWidth === width) {
        return this.cachedLines;
      }
      
      this.cachedLines = this.container.render(width);
      this.cachedWidth = width;
      return this.cachedLines;
    }
    
    invalidate(): void {
      this.cachedWidth = undefined;
      this.cachedLines = undefined;
      this.container.invalidate();
    }
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // TUI COMPONENT: SHORTCUTS OVERLAY
  // ═══════════════════════════════════════════════════════════════════════════
  
  class ShortcutsOverlay {
    private container: Container;
    private cachedWidth?: number;
    private cachedLines?: string[];
    private theme: any;
    
    constructor(theme: any) {
      this.theme = theme;
      this.container = new Container();
      
      // Header
      const header = theme.fg("accent", "╔═══ KEYBOARD SHORTCUTS ═══╗");
      this.container.addChild(new Text(header, 0, 0));
      this.container.addChild(new Spacer(1));
      
      // Navigation section
      const navHeader = theme.fg("warning", "  ⌨️ NAVIGATION");
      this.container.addChild(new Text(navHeader, 0, 0));
      
      const navItems = [
        ["Ctrl+P", "Change model"],
        ["Ctrl+G", "Git status"],
        ["Ctrl+L", "Compact context"],
        ["Ctrl+H", "Help"],
        ["↑↓", "History"],
      ];
      
      for (const [key, action] of navItems) {
        const line = `  ${theme.fg("success", key.padEnd(10))} ${theme.fg("text", action)}`;
        this.container.addChild(new Text(line, 0, 0));
      }
      
      this.container.addChild(new Spacer(1));
      
      // Commands section
      const cmdHeader = theme.fg("warning", "  ⌨️ COMMANDS");
      this.container.addChild(new Text(cmdHeader, 0, 0));
      
      const cmdItems = [
        ["/review", "Code review"],
        ["/component", "Create component"],
        ["/test", "Generate tests"],
        ["/cmd", "Command palette"],
        ["/skill", "Skills"],
      ];
      
      for (const [cmd, action] of cmdItems) {
        const line = `  ${theme.fg("accent", cmd.padEnd(12))} ${theme.fg("text", action)}`;
        this.container.addChild(new Text(line, 0, 0));
      }
      
      this.container.addChild(new Spacer(1));
      
      // OVAV section
      const ovavHeader = theme.fg("warning", "  🔧 OVAV COMMANDS");
      this.container.addChild(new Text(ovavHeader, 0, 0));
      
      const ovavItems = [
        ["/ovav daily", "System status"],
        ["/ovav next", "Next task"],
        ["/ovav validate", "Validate workspace"],
        ["/ovav memory", "Search memories"],
      ];
      
      for (const [cmd, action] of ovavItems) {
        const line = `  ${theme.fg("muted", cmd.padEnd(14))} ${theme.fg("text", action)}`;
        this.container.addChild(new Text(line, 0, 0));
      }
      
      this.container.addChild(new Spacer(1));
      
      // Footer
      const footer = theme.fg("dim", "╚════════════════════════════════╝");
      this.container.addChild(new Text(footer, 0, 0));
    }
    
    handleInput(data: string): void {
      // Any key closes
    }
    
    render(width: number): string[] {
      if (this.cachedLines && this.cachedWidth === width) {
        return this.cachedLines;
      }
      
      this.cachedLines = this.container.render(width);
      this.cachedWidth = width;
      return this.cachedLines;
    }
    
    invalidate(): void {
      this.cachedWidth = undefined;
      this.cachedLines = undefined;
      this.container.invalidate();
    }
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // STATUS BAR — THEME-AWARE
  // ═══════════════════════════════════════════════════════════════════════════
  
  function updateStatusBar(ctx: any) {
    const theme = ctx.ui.theme;
    const model = ctx.model?.id?.split("/").pop() || "none";
    const thinking = ctx.thinkingLevel || "off";
    
    const parts = [
      theme.fg("accent", "🌐"),
      " OVAV",
      theme.fg("muted", "│"),
      theme.fg("text", projectInfo.name),
      theme.fg("muted", "│"),
      theme.fg("success", `🤖 ${model}`),
      theme.fg("muted", "│"),
      theme.fg("warning", `💭 ${thinking}`),
    ];
    
    if (projectInfo.hasChanges) {
      parts.push(theme.fg("muted", "│"));
      parts.push(theme.fg("error", "⚠️"));
    }
    
    ctx.ui.setStatus("ovav-main", parts.join(" "));
  }

  // ═══════════════════════════════════════════════════════════════════════════
  // SESSION START
  // ═══════════════════════════════════════════════════════════════════════════
  
  pi.on("session_start", async (event, ctx) => {
    // DETECT GIT + WORKTREE INFO
    projectInfo = detectGitInfo(ctx.cwd);
    
    // START HOT-RELOAD WATCHER
    const extensionPath = __dirname;
    startHotReloadWatcher(ctx, extensionPath);
    
    // Detect if this is a reload
    const isReload = event.reason === "reload";
    
    if (isReload) {
      ctx.ui.notify("═══════════════════════════════════════", "success");
      ctx.ui.notify("🔄🔄🔄 OVAV PREMIUM INPUT RECARGADO! 🔄🔄🔄", "success");
      ctx.ui.notify("═══════════════════════════════════════", "success");
      ctx.ui.notify("✅ Cambios aplicados - prueba /cmd o ?", "success");
    } else {
      // First load
      ctx.ui.notify("═══════════════════════════════════════", "info");
      ctx.ui.notify("🎉 OVAV PREMIUM INPUT v2.0 CARGADO!", "info");
      ctx.ui.notify("🔥 HOT-RELOAD ACTIVO - Cambios se recargan solos!", "success");
    }
    
    ctx.ui.notify(`📁 Proyecto: ${projectInfo.name}`, "info");
    ctx.ui.notify(`📂 Path: ${projectInfo.path}`, "info");
    
    if (projectInfo.isWorktree) {
      ctx.ui.notify(`🌳 Worktree: ${projectInfo.branch}`, "info");
    } else {
      ctx.ui.notify(`⌥ Rama: ${projectInfo.branch}`, "info");
    }
    
    if (projectInfo.hasChanges) {
      ctx.ui.notify("⚠️ ATENCIÓN: Tienes cambios sin commitear!", "warning");
    }
    
    if (!isReload) {
      ctx.ui.notify("═══════════════════════════════════════", "info");
      ctx.ui.notify("📝 Usa /cmd para paleta de comandos", "info");
      ctx.ui.notify("📝 Usa ? para ver atajos de teclado", "info");
    }
    
    // CUSTOM FOOTER — RICH GIT INFO
    ctx.ui.setFooter((tui, theme, footerData) => {
      const branch = projectInfo.branch;
      const isWorktree = projectInfo.isWorktree;
      const worktreePath = projectInfo.path;
      const changes = projectInfo.hasChanges;
      
      return {
        invalidate() {},
        render(width: number): string[] {
          const worktreeLabel = isWorktree 
            ? `${theme.fg("accent", "🌳")} ${theme.fg("text", "WORKTREE:")} ${theme.fg("muted", branch)}`
            : `${theme.fg("accent", "⌥")} ${theme.fg("text", branch)}`;
          
          const pathText = theme.fg("dim", truncateToWidth(worktreePath, Math.floor(width * 0.4), "..."));
          const changesText = changes ? ` ${theme.fg("error", "⚠️")}` : "";
          
          return [`${worktreeLabel} | ${pathText}${changesText}`];
        },
        dispose: footerData.onBranchChange(() => tui.requestRender()),
      };
    });
    
    // UPDATE STATUS BAR
    updateStatusBar(ctx);
    
    // Working indicator
    ctx.ui.setWorkingIndicator({
      frames: [
        ctx.ui.theme.fg("accent", "◉"),
        ctx.ui.theme.fg("muted", "◎"),
        ctx.ui.theme.fg("accent", "◉"),
        ctx.ui.theme.fg("muted", "◎"),
      ],
      intervalMs: 200,
    });
    
    console.log(`[OVAV Premium v2.0] ${projectInfo.name} | ${projectInfo.branch} | Worktree: ${projectInfo.isWorktree} | Hot-reload: ACTIVE`);
  });

  // ═══════════════════════════════════════════════════════════════════════════
  // SESSION SHUTDOWN — CLEANUP
  // ═══════════════════════════════════════════════════════════════════════════
  
  pi.on("session_shutdown", async (event, ctx) => {
    // Stop hot-reload watcher
    stopHotReloadWatcher();
    
    console.log(`[OVAV Premium] Shutting down (reason: ${event.reason})`);
    
    ctx.ui.setWidget("ovav-status", []);
    ctx.ui.setWidget("ovav-palette", []);
    ctx.ui.setStatus("ovav-main", undefined);
    ctx.ui.setStatus("ovav-turn", undefined);
    ctx.ui.setFooter(undefined);
  });

  // ═══════════════════════════════════════════════════════════════════════════
  // INPUT HANDLING
  // ═══════════════════════════════════════════════════════════════════════════
  
  pi.on("input", async (event, ctx) => {
    const text = event.text || "";
    
    // /cmd - show Command Palette with TUI components
    if (text === "/cmd" || text === "/palette" || text === "/commands") {
      const result = await ctx.ui.custom<string | null>((tui, theme, _kb, done) => {
        const palette = new CommandPalette(theme, getCommandItems(), done, done);
        
        return {
          render: (w) => palette.render(w),
          handleInput: (data) => { palette.handleInput(data); tui.requestRender(); },
          invalidate: () => palette.invalidate(),
        };
      }, { overlay: true });
      
      if (result) {
        ctx.ui.notify(`Selected: ${result}`, "info");
        return { action: "handled" };
      }
      return { action: "handled" };
    }
    
    // ? - show Shortcuts Overlay
    if (text === "?" || text === "/help" || text === "/shortcuts") {
      const result = await ctx.ui.custom<string | null>((tui, theme, _kb, done) => {
        const shortcuts = new ShortcutsOverlay(theme);
        
        return {
          render: (w) => shortcuts.render(w),
          handleInput: (data) => { 
            if (matchesKey(data, Key.escape)) {
              done(null);
            }
            tui.requestRender(); 
          },
          invalidate: () => shortcuts.invalidate(),
        };
      }, { overlay: true });
      
      return { action: "handled" };
    }
    
    // /ovav daily - trigger OVAV daily
    if (text.startsWith("/ovav daily")) {
      ctx.ui.notify("📊 Obteniendo estado diario de OVAV...", "info");
      return { action: "continue" };
    }
  });

  // ═══════════════════════════════════════════════════════════════════════════
  // AGENT EVENTS
  // ═══════════════════════════════════════════════════════════════════════════
  
  pi.on("agent_start", async (_event, ctx) => {
    ctx.ui.setStatus("ovav-main", `${ctx.ui.theme.fg("accent", "🌐")} OVAV | ${ctx.ui.theme.fg("warning", "🔄 Procesando...")}`);
  });

  pi.on("agent_end", async (_event, ctx) => {
    const newInfo = detectGitInfo(ctx.cwd);
    projectInfo = newInfo;
    updateStatusBar(ctx);
  });

  pi.on("turn_start", async (_event, ctx) => {
    ctx.ui.setStatus("ovav-turn", `${ctx.ui.theme.fg("accent", "◐")} Thinking...`);
  });

  pi.on("turn_end", async (_event, ctx) => {
    ctx.ui.setStatus("ovav-turn", undefined);
  });

  // ═══════════════════════════════════════════════════════════════════════════
  // MODEL CHANGE
  // ═══════════════════════════════════════════════════════════════════════════
  
  pi.on("model_select", async (event, ctx) => {
    updateStatusBar(ctx);
    ctx.ui.notify(`${ctx.ui.theme.fg("success", "🤖")} Modelo: ${event.model?.id?.split("/").pop()}`, "info");
  });

  // ═══════════════════════════════════════════════════════════════════════════
  // CUSTOM TOOLS
  // ═══════════════════════════════════════════════════════════════════════════
  
  pi.registerTool({
    name: "ovav_project_info",
    label: "OVAV Project Info",
    description: "Muestra información del proyecto y worktree actual",
    parameters: Type.Object({}),
    async execute(_id, _params, _signal, _onUpdate, ctx) {
      projectInfo = detectGitInfo(ctx.cwd);
      
      return {
        content: [{
          type: "text",
          text: [
            "═══════════════════════════════════════════════",
            "📊 OVAV PROJECT INFO",
            "═══════════════════════════════════════════════",
            `📁 Proyecto: ${projectInfo.name}`,
            `📂 Path: ${projectInfo.path}`,
            `⌥ Rama: ${projectInfo.branch}`,
            projectInfo.isWorktree ? `🌳 Tipo: WORKTREE` : `📂 Tipo: REPOSITORIO PRINCIPAL`,
            projectInfo.isWorktree ? `🏠 Root: ${projectInfo.worktreeRoot}` : "",
            projectInfo.hasChanges ? "⚠️ Cambios sin commitear" : "✓ Sin cambios pendientes",
            "═══════════════════════════════════════════════",
          ].filter(Boolean).join("\n")
        }],
        details: projectInfo,
      };
    },
  });

  // ═══════════════════════════════════════════════════════════════════════════
  // COMMANDS
  // ═══════════════════════════════════════════════════════════════════════════
  
  pi.registerCommand("ovav-status", {
    description: "Show OVAV project status",
    handler: async (_args, ctx) => {
      projectInfo = detectGitInfo(ctx.cwd);
      updateStatusBar(ctx);
      ctx.ui.notify(`📊 Status: ${projectInfo.name} | ${projectInfo.branch}`, "info");
    },
  });

  pi.registerCommand("cmd", {
    description: "Show command palette (TUI)",
    handler: async (_args, ctx) => {
      const result = await ctx.ui.custom<string | null>((tui, theme, _kb, done) => {
        const palette = new CommandPalette(theme, getCommandItems(), done, done);
        
        return {
          render: (w) => palette.render(w),
          handleInput: (data) => { palette.handleInput(data); tui.requestRender(); },
          invalidate: () => palette.invalidate(),
        };
      }, { overlay: true });
      
      if (result) {
        ctx.ui.notify(`Selected: ${result}`, "info");
      }
    },
  });

  console.log("[OVAV Premium Input v2.0] TUI Components + HOT-RELOAD loaded");
}
