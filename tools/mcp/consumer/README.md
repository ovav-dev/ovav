# OVAV Windows MCP services

OpenCode uses two Windows singleton MCP services over loopback. The Atuin MCP
process is disabled in WSL; Atuin shell/history remains installed.

| Service | Endpoint | Windows implementation |
|---|---|---|
| `ovav-playwright` | `http://127.0.0.1:8931/mcp` | Node 22 + exact Playwright MCP, Chrome, managed profile |
| `ovav-memory` | `http://127.0.0.1:8932/mcp` | exact supergateway + memory MCP, stateful Streamable HTTP |

Both servers bind `127.0.0.1` only. Playwright also receives
`--allowed-hosts 127.0.0.1,localhost`. Authentication is unnecessary only
under the stated threat model: loopback sockets are not remotely reachable,
and local code execution is already trusted. Never expose these ports on a
non-loopback interface.

## Integrity-pinned installation

`bin/windows-bundle/package.json` pins exact direct versions and the reviewed
`package-lock.json` pins the full dependency graph with registry integrity
hashes. The manager copies both manifests to the versioned stable bundle and
runs only:

```text
npm ci --ignore-scripts --no-audit --no-fund
```

There is no mutable `npm install` fallback. Installation fails closed if the
lock is absent, malformed, lacks SHA-512 registry integrity, or resolves a
different direct version.

## Lifecycle

Run only with approval for Windows current-user writes:

```powershell
pwsh -NoProfile -File .\tools\mcp\consumer\bin\ovav-mcp-windows.ps1 -Mode Install
pwsh -NoProfile -File "$env:LOCALAPPDATA\OVAV\mcp\manager\1.1.0\ovav-mcp-windows.ps1" -Mode Start
```

`Install`, `Start`, `Stop`, `Recover`, and `Uninstall` share a cross-process
exclusive mutex. The manager validates the full `%LOCALAPPDATA%` ancestor
chain for reparse points, creates private current-user ACL directories, and
revalidates immediately before every destructive leaf operation. It never
recursively deletes `data`.

State is atomically replaced and guarded by schema + SHA-256 checksum through
`intent`, `starting`, `running`, `stopping`, and `recovery-required` phases.
`Status` fails closed on corruption. `Recover` kills only durable, exactly
verified identities; corrupt state recovery kills nothing and proceeds only
when both managed ports are unoccupied.

The scheduled task has a collision-resistant name and is changed or removed
only when its executable, exact arguments, current-user principal, and stable
manager path match. No `ExecutionPolicy Bypass` is used.

## Memory service and data

The Windows Memory MCP service maintains its own state at
`%LOCALAPPDATA%\OVAV\mcp\data\memory.jsonl`. The OVAV memory knowledge graph
remains on WSL; memory is not migrated between environments. Uninstall preserves
everything under `%LOCALAPPDATA%\OVAV\mcp\data`.

The memory gateway session timeout is 300000 ms. Health requires an owned
listener plus successful MCP `initialize` and `tools/list`; `/healthz` alone
is not accepted.

## Tests

Only PowerShell 7 is accepted:

```powershell
pwsh -NoLogo -NoProfile -NonInteractive `
  -File .\tools\mcp\consumer\test\ovav-mcp-windows-manager.Tests.ps1
```

The executable fixtures cover task ownership, reparse rejection, PID reuse,
exact command mismatch, corrupt state, partial start, root-dead recorded child,
port-owner mismatch, data preservation, parser validity, and mutex exclusion.

After changing `opencode.json`, fully restart OpenCode; configuration is not
hot-reloaded.
