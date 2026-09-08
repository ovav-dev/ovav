#!/usr/bin/env bash
# OVAV clipboard bridge — text + image clipboard for WSL2/Linux/macOS hosts.
#
# Replaces xclip/win32yank usage from OVAV shell aliases. Acts as a single
# binary that:
#   * reads text from clipboard (-o / --output, default type=auto)
#   * writes text to clipboard (stdin, default)
#   * extracts image bytes from clipboard (--type=image) and saves them to a
#     managed inbox at $OVAV_CLIPBOARD_INBOX (default ~/.local/share/ovav/clipboard/inbox),
#     emitting the resulting path on stdout so downstream tools (opencode TUI,
#     scripts) can attach/reference it as @/path.
#
# Host detection:
#   * WSL2       -> win32yank.exe for text, powershell.exe + Get-Clipboard
#                   for image MIME (since win32yank is text-only).
#   * Linux X11  -> xclip
#   * Linux Wayland -> wl-paste
#   * macOS      -> pngpaste / osascript fallback
#
# Exit codes:
#   0 success, 1 unsupported type for host, 2 no clipboard tool, 3 read empty.

set -euo pipefail

OVAV_CLIPBOARD_INBOX="${OVAV_CLIPBOARD_INBOX:-$HOME/.local/share/ovav/clipboard/inbox}"
OVAV_BRIDGE_VERSION="3.0.0"

# ── Host detection ────────────────────────────────────────────────────────────
is_wsl() { grep -qi 'microsoft\|wsl' /proc/version 2>/dev/null; }
has() { command -v "$1" >/dev/null 2>&1; }

session_type() {
    if is_wsl; then echo wsl; return; fi
    if [ -n "${WAYLAND_DISPLAY:-}" ] && has wl-paste; then echo wayland; return; fi
    if [ -n "${DISPLAY:-}" ] && has xclip; then echo x11; return; fi
    if [ "$(uname -s)" = "Darwin" ]; then echo macos; return; fi
    echo unknown
}

ensure_inbox() {
    mkdir -p "$OVAV_CLIPBOARD_INBOX"
}

# ── Text operations ───────────────────────────────────────────────────────────
text_read() {
    case "$(session_type)" in
        wsl)     win32yank.exe -o --lf ;;
        x11)     xclip -selection clipboard -o ;;
        wayland) wl-paste --no-clipboard ;;
        macos)   pbpaste ;;
        *)       echo "ovav-clipboard-bridge: no clipboard tool for host" >&2; return 2 ;;
    esac
}

text_write() {
    case "$(session_type)" in
        wsl)     win32yank.exe -i ;;
        x11)     xclip -selection clipboard -in ;;
        wayland) wl-copy ;;
        macos)   pbpaste >/dev/null; pbcopy ;;
        *)       echo "ovav-clipboard-bridge: no clipboard tool for host" >&2; return 2 ;;
    esac
}

# ── Image operations ──────────────────────────────────────────────────────────
# Detect available MIME targets on the clipboard. Returns a newline-separated list.
image_targets() {
    case "$(session_type)" in
        wsl)
            powershell.exe -NoProfile -Command \
                'Add-Type -AssemblyName System.Windows.Forms; $img = [System.Windows.Forms.Clipboard]::GetImage(); if ($img) { "image/png" } else { "" }' \
                2>/dev/null | tr -d '\r' | grep -E '^image/' || true
            ;;
        x11)
            xclip -selection clipboard -t TARGETS -o 2>/dev/null \
                | tr -s ' ' '\n' | grep -E '^image/' || true
            ;;
        wayland)
            wl-paste --list-types 2>/dev/null | grep -E '^image/' || true
            ;;
        macos)
            osascript -e 'clipboard info' 2>/dev/null | tr ',' '\n' | grep -E '^TIFF' || true
            ;;
        *) return 1 ;;
    esac
}

# Pick the best image MIME available. Echoes chosen MIME or empty string.
pick_image_mime() {
    local targets
    targets="$(image_targets || true)"
    if [ -z "$targets" ]; then echo ""; return; fi
    # Preference order
    for m in image/png image/jpeg image/gif image/webp image/bmp image/tiff; do
        if printf '%s\n' "$targets" | grep -qx "$m"; then
            echo "$m"
            return
        fi
    done
    # Fallback to first image/* offered
    printf '%s\n' "$targets" | head -1
}

# Extension for a given MIME.
mime_to_ext() {
    case "$1" in
        image/png)  echo png ;;
        image/jpeg) echo jpg ;;
        image/gif)  echo gif ;;
        image/webp) echo webp ;;
        image/bmp)  echo bmp ;;
        image/tiff) echo tif ;;
        *)          echo bin ;;
    esac
}

# Read image bytes from clipboard into the given file path. Returns 0 on success.
image_extract_to() {
    local out_path="$1"
    local mime="${2:-}"
    if [ -z "$mime" ]; then mime="$(pick_image_mime)"; fi
    if [ -z "$mime" ]; then return 3; fi

    case "$(session_type)" in
        wsl)
            # powershell writes the image bytes to stdout. We use a base64 detour
            # because powershell -> WSL pipe can mangle binary.
            local b64
            b64="$(powershell.exe -NoProfile -Command \
                'Add-Type -AssemblyName System.Windows.Forms; $img = [System.Windows.Forms.Clipboard]::GetImage(); if ($img) { $ms = New-Object System.IO.MemoryStream; $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png); [Convert]::ToBase64String($ms.ToArray()) }' \
                2>/dev/null | tr -d '\r\n' )"
            if [ -z "$b64" ]; then return 3; fi
            printf '%s' "$b64" | base64 -d > "$out_path"
            ;;
        x11)
            xclip -selection clipboard -t "$mime" -o > "$out_path" 2>/dev/null
            ;;
        wayland)
            wl-paste --type "$mime" > "$out_path" 2>/dev/null
            ;;
        macos)
            # Read TIFF (the only image type macOS exposes) and convert via sips.
            local tiff
            tiff="$(mktemp -t ovav-clip).tiff"
            osascript -e 'the clipboard as «class PNGf»' 2>/dev/null > "$tiff" \
                || pngpaste "$tiff" 2>/dev/null \
                || { rm -f "$tiff"; return 3; }
            sips -s format png "$tiff" --out "$out_path" >/dev/null 2>&1
            rm -f "$tiff"
            ;;
        *) return 1 ;;
    esac
}

# Public: read clipboard image to managed inbox, output path.
image_read() {
    ensure_inbox
    local mime
    mime="$(pick_image_mime)"
    if [ -z "$mime" ]; then return 3; fi
    local ext
    ext="$(mime_to_ext "$mime")"
    local stamp
    stamp="$(date -u +%Y%m%dT%H%M%SZ)"
    local out="$OVAV_CLIPBOARD_INBOX/$stamp.$ext"
    if image_extract_to "$out" "$mime"; then
        if [ -s "$out" ]; then
            printf '%s\n' "$out"
            return 0
        fi
        rm -f "$out"
    fi
    return 3
}

# ── CLI parsing ──────────────────────────────────────────────────────────────
mode="write"   # write | read
type="auto"    # auto | text | image

while [ $# -gt 0 ]; do
    case "$1" in
        -V|-version|--version)
            printf 'ovav-clipboard-bridge %s\n' "$OVAV_BRIDGE_VERSION"
            exit 0 ;;
        -o|--out|--output) mode="read"; type="auto"; shift ;;
        --type=*) type="${1#--type=}"; shift ;;
        --type)  type="${2:-auto}"; shift 2 ;;
        -h|--help)
            cat <<'EOF'
ovav-clipboard-bridge — text + image clipboard bridge

Usage:
  ovav-clipboard-bridge              Write stdin to clipboard as text.
  ovav-clipboard-bridge -o           Read clipboard (auto text|image) to stdout.
                                     If image: saves to inbox and prints path.
  ovav-clipboard-bridge -o --type=text   Force text read.
  ovav-clipboard-bridge -o --type=image  Force image extract.

Environment:
  OVAV_CLIPBOARD_INBOX  Where image extracts land (default ~/.local/share/ovav/clipboard/inbox).

Examples:
  # Type a clipboard image into opencode TUI as @path:
  ovav-clipboard-bridge -o --type=image | xargs -I{} opencode -a {}

  # Inbox the image, then attach by path:
  img=$(ovav-clipboard-bridge -o --type=image)
  echo "Read $img"
EOF
            exit 0 ;;
        *) echo "ovav-clipboard-bridge: unknown arg: $1" >&2; exit 2 ;;
    esac
done

case "$mode:$type" in
    read:auto)
        # Probe: prefer image if present and non-empty.
        if [ -n "$(pick_image_mime)" ]; then
            image_read || { text_read; }
        else
            text_read
        fi
        ;;
    read:text)  text_read ;;
    read:image) image_read ;;
    write:*)
        text_write
        ;;
esac
