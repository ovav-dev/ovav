# Shell Integration for Intelligent Terminal — bash
# Emits OSC 133 (command marks / exit code) and OSC 9;9 (CWD) sequences
# WITHOUT altering the visual appearance of the user's prompt.
#
# Compatible with bash 3.2+. Safe to source multiple times.
# Silently no-ops outside Intelligent Terminal, in non-interactive shells,
# and in non-bash shells.
# Every variable read uses ${VAR:-} defaulting so the script is safe
# even when the user has `set -u` earlier in their .bashrc.

# Guard: Intelligent Terminal host, bash only, interactive only, idempotent.
[ "${INTELLIGENT_TERMINAL:-}" = "1" ] || return 0 2>/dev/null
[ -z "${BASH_VERSION:-}" ] && return 0 2>/dev/null
case "${-:-}" in *i*) ;; *) return 0 2>/dev/null ;; esac
[ -n "${__IT_SHELLINTEG_INSTALLED:-}" ] && return 0 2>/dev/null
__IT_SHELLINTEG_INSTALLED=1

# Snapshot the user's PROMPT_COMMAND once; we re-run it from our wrapper
# so we don't clobber any existing hook (starship, oh-my-bash, etc).
# Walk the array element-by-element and join the NON-EMPTY entries
# with ';'. Joining with IFS via ${ARR[*]} would include empty array
# slots verbatim, producing leading or embedded ';' tokens that eval
# rejects with "syntax error near unexpected token \;'". Ubuntu's
# /etc/profile.d/80-systemd-osc-context.sh deliberately reserves
# PROMPT_COMMAND[0]='' before appending its hook, and that empty slot
# was the trigger.
#
# "${PROMPT_COMMAND[@]}" expands to one element on a scalar var, zero
# elements when unset, and N elements on an array -- the same loop
# handles bash 3.2+ scalar and bash 5.1+ array uniformly. The
# ${PROMPT_COMMAND[@]+...} alternate-value guard keeps it `set -u`
# safe: when PROMPT_COMMAND is unset the whole expansion collapses to
# nothing instead of tripping "unbound variable" on bash 3.2 (a plain
# "${PROMPT_COMMAND[@]}" errors there). ${PROMPT_COMMAND:-} can't be
# used: scalar ${VAR:-} defaulting only sees element [0] of an array,
# which is exactly the empty-slot case this loop exists to handle.
__IT_SHELLINTEG_USER_PC=""
for __it_pc_entry in ${PROMPT_COMMAND[@]+"${PROMPT_COMMAND[@]}"}; do
    [ -z "$__it_pc_entry" ] && continue
    if [ -z "$__IT_SHELLINTEG_USER_PC" ]; then
        __IT_SHELLINTEG_USER_PC="$__it_pc_entry"
    else
        __IT_SHELLINTEG_USER_PC="$__IT_SHELLINTEG_USER_PC;$__it_pc_entry"
    fi
done
unset __it_pc_entry

__it_shellinteg_prompt() {
    local __ec=$?
    local __it_b=$'\033]133;B\007'
    local __it_suffix="\[${__it_b}\]"

    # Finish the previous command before running prompt hooks.
    printf '\033]133;D;%s\007' "$__ec"

    # Remove our previous suffix before the user's hook sees PS1. Prompt
    # frameworks may then freely rebuild PS1 before we append a fresh suffix.
    case "${PS1:-}" in
        *"$__it_suffix") PS1="${PS1%"$__it_suffix"}" ;;
    esac

    if [ -n "$__IT_SHELLINTEG_USER_PC" ]; then
        # Restore $? for the user's PROMPT_COMMAND so hooks like
        # `local ec=$?` at its top still see the real exit code
        # instead of printf's success status.
        (exit "$__ec"); eval "$__IT_SHELLINTEG_USER_PC"
    fi

    # Append OSC 133;B (command-input start) to the final PS1. The \[ \]
    # brackets tell readline these bytes are zero-width. Re-check PS1 below
    # before emitting OSC 133;A so a failed assignment cannot open an
    # unterminated semantic-prompt region.
    case "${PS1:-}" in
        *"$__it_suffix") ;;
        *) PS1="${PS1:-}${__it_suffix}" ;;
    esac

    # OSC 9;9 unquoted form (no surrounding double quotes around the
    # path). Linux paths can contain `"` (only `/` and NUL are
    # forbidden), and Terminal's 9;9 parser rejects the quoted form
    # when the payload contains embedded quotes — silently dropping
    # CWD reporting for those directories. The unquoted form parses
    # cleanly regardless of path contents.
    printf '\033]9;9;%s\007' "${PWD:-}"
    # OSC 9001;ShellType — report shell identity each prompt so the terminal
    # always knows which shell owns the pane, even after a nested shell exits.
    # Under WSL, $WSL_DISTRO_NAME is set so we report "wsl:<distro>"; plain
    # (Git) bash reports "bash".
    if [ -n "${WSL_DISTRO_NAME:-}" ]; then
        printf '\033]9001;ShellType;wsl:%s;%s\007' "$WSL_DISTRO_NAME" "${BASH_VERSION:-}"
    else
        printf '\033]9001;ShellType;bash;%s\007' "${BASH_VERSION:-}"
    fi

    # Open a prompt region only when Bash is guaranteed to close it by
    # expanding the OSC 133;B suffix from PS1 immediately after this hook.
    case "${PS1:-}" in
        *"$__it_suffix") printf '\033]133;A\007' ;;
    esac
}
PROMPT_COMMAND=__it_shellinteg_prompt
