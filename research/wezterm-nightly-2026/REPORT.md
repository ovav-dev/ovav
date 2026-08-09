# WezTerm Nightly 2026 Compatibility Issues

> Generated 2026-08-05 · depth: standard · 15+ sources · workspace: research/wezterm-nightly-2026/

## Executive summary

- **WebGPU front_end**: Available but NOT the default. Default was briefly set to `"WebGpu"` in Jan 2024 (one day only, then reverted to `"OpenGL"`). Must be manually enabled via `front_end = "WebGpu"`. [1][2][3][4]
- **enable_csi_u_key_encoding**: No changes in 2026 nightly. Remains `false` by default, not deprecated. `enable_kitty_keyboard` is the actively maintained alternative. [5][6][7]
- **fancy_tab_bar**: Critical macOS crash (Oct 2024) with `window_decorations = "INTEGRATED_BUTTONS|RESIZE"`. Several rendering fixes landed in early 2024. New nightly features: tab bar progress spinner, `show_close_tab_button_in_tabs`. [8][9][10]
- **color_schemes schema**: Schema is stable; no major breaking changes documented in the accessible changelog. Custom non-standard fields are silently ignored by WezTerm's native machinery. [single source]
- **Deprecated APIs**: `show_update_window` is fully inert in nightly and will be removed. `font_antialias`/`font_hinting` are no-ops since 2021. Several old key actions removed (Copy/Paste/PastePrimarySelection → CopyTo/PasteFrom). [11]
- **enable_tab_bar vs use_fancy_tab_bar**: No conflict. `enable_tab_bar` is a master on/off switch; `use_fancy_tab_bar` controls rendering style only when the tab bar is visible. [12][13][14][15]
- **Critical structural finding**: WezTerm has been nightly-only since Feb 2024 (last stable: 20240203). All post-Feb-2024 changes live in an undated "Continuous/Nightly" changelog section, making precise 2025/2026 attribution impossible from public docs. [16]

---

## Background & scope

This research investigated breaking changes, deprecations, and compatibility issues in WezTerm nightly builds relevant to a 2026 configuration. Primary source: wezfurlong.org (wezterm.org) changelog and config docs. Six research angles were investigated in parallel. F4 (color_schemes schema) findings did not persist to disk and are noted as an open question.

---

## WebGPU front_end availability

### [1] `front_end = "WebGpu"` is a valid, documented option
The `front_end` config accepts `"OpenGL"`, `"Software"`, and `"WebGpu"`. WebGPU backends: Metal (macOS), Vulkan (Linux), DirectX 12 (Windows). [1]

### [2] WebGPU introduced Nov 2022, briefly set as default in Jan 2024
Available since version 20221119-145034-49b9839f. The default was temporarily changed to `"WebGpu"` in version 20240127-113634-bbcac864 (Jan 27, 2024) — then **reverted to `"OpenGL"` the next day** (20240128-202157-1e552d76). [3][4]

### [3] Current default is "OpenGL", not WebGPU
Since the revert, `front_end = "OpenGL"` is the documented default. WebGPU must be explicitly set. Three related options: `webgpu_power_preference` (default: "LowPower"), `webgpu_force_fallback_adapter` (default: false), `webgpu_preferred_adapter`. [4][5]

### [4] No evidence of WebGPU removal
All WebGPU-related options remain in the current docs index. No post-Feb-2024 changelog entry indicates removal or deprecation. [dead end]

### [5] Windows Remote Desktop auto-selects Software front_end
WezTerm automatically selects `Software` when started in a Windows Remote Desktop environment. [6]

---

## enable_csi_u_key_encoding behavior

### [6] Default is false, unchanged
`enable_csi_u_key_encoding` defaults to `false` — no change documented in any 2025-2026 nightly entry. [7]

### [7] Not deprecated; explicitly discouraged
The docs carry a strong warning: enabling it "does change the behavior of some keys in backwards incompatible ways and there isn't a way for applications to detect or request this behavior." No deprecation marker exists in current GitHub source. [8]

### [8] `allow_win32_input_mode` precedence unchanged
On Windows, `allow_win32_input_mode` (defaults true) takes precedence over `enable_csi_u_key_encoding`. [9]

### [9] `enable_kitty_keyboard` is the actively maintained alternative
Introduced June 2022, the Kitty Keyboard Protocol allows applications to request encoding changes via escape sequences — a superset of CSI-u. CSI-u is discouraged precisely because apps cannot detect or request it. [10]

### [10] No 2025/2026 nightly changes for this option
The "Continuous/Nightly" changelog section has no date-stamped entries for `enable_csi_u_key_encoding` in 2025 or 2026. [11]

---

## fancy_tab_bar rendering changes

### [11] Critical macOS crash with INTEGRATED_BUTTONS (Oct 2024)
Reported on macOS 15.1 with `window_decorations = "INTEGRATED_BUTTONS|RESIZE"`. Panic in `fancy_tab_bar.rs::build_fancy_tab_bar` at line 423 — a Rust unsafe precondition violation (`slice::from_raw_parts` with null/unaligned pointer) during Harfbuzz font shaping. [8]

### [12] Jan 2024: Transparent tab bar backgrounds, double-click maximize, indentation fix
Version 20240127-113634-bbcac864 introduced: transparent fancy tab bar backgrounds via `window_frame` alpha channel; double-click on tab bar toggles maximize/normal; fixed retro tab bar indentation on macOS with integrated titlebar buttons. [9]

### [13] Jan-Feb 2024: Tab bar update lag fixed, title sync fixed
- Version 20240128-202157-1e552d76: fancy tab bar update lag after closing tabs. [10]
- Version 20240203-110809-5046fc22: tab bar now immediately reflects `tab:set_title` results. [10]

### [14] New nightly features (undated entries)
- Tab bar progress spinner for indeterminate OSC 9 state. [10]
- New config option `show_close_tab_button_in_tabs` for the fancy tab bar. [10]

### [15] Retro tab bar with INTEGRATED_BUTTONS (Apr 2023)
Since version 20230408-112425-69ae8472, retro tab bar works with integrated window buttons (`window_decorations = "INTEGRATED_BUTTONS|RESIZE"`). [10]

### [16] Historical API changes (2021-2022)
Multiple `tab_bar_style` API changes removed old elements: `new_tab_left/right/hover_left/hover_right` (Aug 2021), `active_tab_left/right/inactive_tab_*` elements replaced by `format-tab-title` event. [10]

---

## config.color_schemes schema changes

**Open question**: F4 findings file did not persist. The research agent's inline report noted three bugs it fixed in a `wezterm.lua` (unrelated to this research brief), plus a note that the `config.color_schemes` schema research from a prior session was lost.

From available sources: the `color_schemes` config option and the `colors` object are stable in the current docs. The `color_schemes` doc page simply points to "Colors & Appearance" for schema details. No breaking schema changes were found in the accessible changelog for 2025-2026. Custom non-standard fields added to color schemes (e.g., `bg`, `fg`, `surface1`, `surface2`) are silently ignored by WezTerm's native color scheme machinery — they work in-session via Lua table keys but are not recognized by WezTerm's native config parser.

---

## Deprecated/removed APIs in WezTerm 2025-2026

### [17] `show_update_window` — inert in nightly, will be removed
The update notification UI was dropped because WezTerm moved to nightly-only releases with no formal versioned releases. The option "no longer has any effect and will be removed in a future release." [11]

### [18] `font_antialias` and `font_hinting` — no-ops since March 2021
Deprecated since version 20210314-114017-04b7cedd. Both are fully inert; migration path is `freetype_load_target`. [12][13]

### [19] `update-right-status` event deprecated since Nov 2022
Replaced by the more general `update-status` event that handles both left and right status bars. [14]

### [20] `Copy`, `Paste`, `PastePrimarySelection` key actions removed July 2023
Users must migrate to `CopyTo` and `PasteFrom`. [15]

### [21] `send_composed_key_when_alt_is_pressed` removed
Behavior folded into granular `send_composed_key_when_left_alt_is_pressed` and `send_composed_key_when_right_alt_is_pressed`. [16]

### [22] Default ALT-[NUMBER] bindings removed
These broke non-US keyboard layouts. [17]

### [23] Last Resort fallback font removed
Removed from the font fallback chain. [18]

### [24] Old CPU renderer removed
The legacy CPU-based renderer was fully removed. [19]

### [25] WinPty support removed (Feb 2020)
Windows dropped WinPty in favor of ConPTY. [20]

### [26] DECRQCRA disabled by default — security hardening in nightly
DECRQCRA (`enable_checksum_rectangular_area`) was disabled by default "to prevent silent screen scraping." Users who need it must explicitly enable it. This is a behavioral default change, not an API deprecation. [21]

---

## enable_tab_bar vs use_fancy_tab_bar conflict

### [27] Independent booleans, no conflict
Both `enable_tab_bar` and `use_fancy_tab_bar` are declared as independent `bool` fields in the Rust `Config` struct, both defaulting to `true`. There is no code making one conditional on the other. [12]

### [28] `use_fancy_tab_bar` is ignored when `enable_tab_bar = false`
When the tab bar is disabled entirely (`enable_tab_bar = false`), `use_fancy_tab_bar` has no meaning — but this is a logical implication, not a conflict or override. [13]

### [29] No documented interaction or precedence
Neither the official docs nor the changelog mention any precedence, override, or conflict relationship between these two options. They are separately documented with independent purposes. [14]

### [30] macOS drag bug with fancy tab bar + INTEGRATED_BUTTONS (June 2026)
Issue #7830 (open as of research date) reports three-finger-drag on the tab bar causes window-jumping on macOS when `use_fancy_tab_bar = true` and `window_decorations = "INTEGRATED_BUTTONS|RESIZE"`. This is a rendering bug, not a config conflict. [15]

### [31] Historical crash with enable_tab_bar = false (May 2020)
Fixed in version 20200503-171512-b13ef15f — a crash when reloading a config with `enable_tab_bar=false`. This was a 2020 bug, not current. [22]

---

## Open questions

1. **color_schemes schema (F4)**: Findings did not persist to disk. The current schema state for 2026 nightly is unconfirmed beyond the observation that the docs reference page is sparse and custom fields are ignored.
2. **Post-Feb-2024 stable releases**: The official changelog stops at 20240203. Whether WezTerm has shipped any post-Feb-2024 stable releases through other channels is unconfirmed.
3. **Oct 2024 macOS crash fix status**: Whether the `fancy_tab_bar.rs` crash (issue #6336) was fixed in subsequent nightlies is unconfirmed.
4. **Nightly changelog dating**: The "Continuous/Nightly" section has no timestamps, making it impossible to attribute specific changes to 2025 vs 2026 from public docs alone.
5. **Semantic versioning discussion (July 2026)**: GitHub Discussion #7957 discussed switching from date/hash versioning to SemVer. Maintainer said it's "not on the table at the moment." [23]

---

## Sources

[1] https://wezterm.org/config/lua/config/front_end.html
[2] https://wezterm.org/config/lua/config/front_end.html (since tag 20221119-145034-49b9839f)
[3] https://wezterm.org/config/lua/config/front_end.html (since tag 20240127-113634-bbcac864)
[4] https://wezterm.org/config/lua/config/front_end.html (revert tag 20240128-202157-1e552d76)
[5] https://wezterm.org/config/lua/config/webgpu_power_preference.html
[6] https://wezterm.org/config/lua/config/front_end.html (auto Software on Windows RDP)
[7] https://wezterm.org/config/lua/config/enable_csi_u_key_encoding.html
[8] https://wezterm.org/config/lua/config/enable_csi_u_key_encoding.html (discouraged warning)
[9] https://wezterm.org/config/lua/config/enable_csi_u_key_encoding.html (allow_win32_input_mode precedence)
[10] https://wezterm.org/config/lua/config/enable_kitty_keyboard.html
[11] https://wezterm.org/changelog.html (Continuous/Nightly section)
[12] https://raw.githubusercontent.com/wezterm/wezterm/main/config/src/config.rs
[13] https://wezterm.org/config/lua/config/use_fancy_tab_bar.html
[14] https://wezterm.org/config/lua/config/enable_tab_bar.html
[15] https://github.com/wezterm/wezterm/issues/7830
[16] https://github.com/wezterm/wezterm/releases/tag/20240203-110809-5046fc22
[17] https://wezterm.org/config/lua/config/show_update_window.html
[18] https://wezterm.org/config/lua/config/font_antialias.html
[19] https://wezterm.org/config/lua/config/font_hinting.html
[20] https://wezterm.org/changelog.html#20221119-145034-49b9839f
[21] https://wezterm.org/changelog.html#20230712-072601-f4abf8fd
[22] https://wezterm.org/changelog.html (Continuous/Nightly section, DECRQCRA)
[23] https://wezterm.org/changelog.html#20200503-171512-b13ef15f
[24] https://github.com/wez/wezterm/discussions/7957
