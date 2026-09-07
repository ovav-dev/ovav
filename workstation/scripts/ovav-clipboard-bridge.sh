#!/usr/bin/env bash
# xclip-compatible bridge for WSL2 using win32yank
# win32yank.exe lives at ~/.local/bin/ and works reliably with WSL2 clipboard
set -euo pipefail

is_wsl() {
    grep -qi 'microsoft\|wsl' /proc/version 2>/dev/null
}

read_clipboard() {
    if is_wsl; then
        win32yank.exe -o --lf
    else
        if command -v xclip.real >/dev/null 2>&1; then
            exec xclip.real -selection clipboard -o
        elif command -v xclip >/dev/null 2>&1; then
            exec xclip -selection clipboard -o
        fi
        echo "xclip: no se pudo acceder al clipboard" >&2; exit 1
    fi
}

write_clipboard() {
    if is_wsl; then
        win32yank.exe -i
    else
        if command -v xclip.real >/dev/null 2>&1; then
            exec xclip.real -selection clipboard -
        elif command -v xclip >/dev/null 2>&1; then
            exec xclip -selection clipboard -
        fi
        echo "xclip: no se pudo escribir al clipboard" >&2; exit 1
    fi
}

read_mode=0
for arg in "$@"; do
    case "$arg" in
        -version|-V)
            printf '%s\n' 'ovav-xclip-bridge 2.5 (WSL2 win32yank)'
            exit 0
            ;;
        -o|--out|--output) read_mode=1 ;;
    esac
done

if ((read_mode)); then
    read_clipboard
else
    write_clipboard
fi
