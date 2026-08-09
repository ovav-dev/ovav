# ──────────────────────────────────────────────────────────────────────
# OVAV OWS Shell Wrappers — FISH native version
# Source: source /home/braka/Systems/OVAV/bin/ovav-ow-shim.fish
# ──────────────────────────────────────────────────────────────────────

if not set -q OVAV_ROOT
    set -gx OVAV_ROOT /home/braka/Systems/OVAV
end
if not set -q OVAV_CONSUMER_TIER
    set -gx OVAV_CONSUMER_TIER business
end
if not set -q OVAV_CONSUMER_ID
    set -gx OVAV_CONSUMER_ID (whoami)
end

# Colors
set -g _CY (set_color cyan)
set -g _GR (set_color green)
set -g _RE (set_color red)
set -g _DM (set_color brblack)
set -g _BD (set_color --bold)
set -g _RS (set_color normal)

# Dispatch: silent execution + clean result
function _ow_dispatch
    set -l cmd $argv[1]
    set -l args $argv[2..-1]
    set -gx OVAV_CONSUMER_ROOT $PWD
    set -l _cd_target ""
    set -l _error ""
    set -l _branch ""
    set -l _action ""

    switch $cmd
        case c;    set _action "creating worktree"
        case d;    set _action "finalizing"
        case l;    set _action "listing"
        case v;    set _action "verifying"
        case s;    set _action "syncing"
        case clean; set _action "cleaning"
        case m;    set _action "moving"
        case x;    set _action "cherry-picking"
        case a;    set _action "aborting"
        case r;    set _action "rescuing"
        case lk;   set _action "locking"
        case '*';  set _action "running"
    end

    # Spinner: single row, no newline
    set -l frames "⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏"
    set -l fi 0

    # Run command silently, capture PWD_NOW + errors
    $OVAV_ROOT/bin/ovav-ow-runtime.sh $cmd $args 2>&1 | while read -l rawline
        set -l line (string replace -r '\r' '' -- $rawline)
        string length -q -- "$line"; or continue

        # Update spinner
        set fi (math "($fi + 1) % "(count $frames))
        printf "\r  %s %s" $frames[(math $fi + 1)] "$_action"

        switch $line
            case 'PWD_NOW=*'
                set _cd_target (string replace -r '^PWD_NOW=' '' -- $line)
            case '*ERROR*' '*BLOCKED*' '*denied*'
                set _error "$line"
            case '[ovav-owc created]*' '*creado*'
                set _branch (string replace -r '^\[.*\]\s*' '' -- $line)
        end
    end

    # Clear spinner line
    printf "\r%s\r" (string repeat -n 40 " ")

    # Final result
    if test -n "$_error"
        printf "\n%s✖ %s%s\n" "$_RE" "$_error" "$_RS"
    else if test -n "$_branch"
        set -l short_path (string replace -r '.*/OVAV-' '' -- "$_cd_target")
        printf "\n%s✔ %s → %s%s\n" "$_GR" (string upper -- "$_branch") "$_RS" "$_DM" "$short_path" "$_RS"
    else
        printf "\n%s✔ Done%s\n" "$_GR" "$_RS"
    end

    # Real cd
    if test -n "$_cd_target" -a -d "$_cd_target"
        cd "$_cd_target"
    end
end

# Per-command wrappers
function owc;    _ow_dispatch c     $argv; end
function owd;    _ow_dispatch d     $argv; end
function owl;    _ow_dispatch l     $argv; end
function owv;    _ow_dispatch v     $argv; end
function ows;    _ow_dispatch s     $argv; end
function owclean; _ow_dispatch clean $argv; end
function owm;    _ow_dispatch m     $argv; end
function owx;    _ow_dispatch x     $argv; end
function owa;    _ow_dispatch a     $argv; end
function owr;    _ow_dispatch r     $argv; end
function owlk;   _ow_dispatch lk    $argv; end
function owprep;     _ow_dispatch prep    $argv; end
function owsuggest;  _ow_dispatch suggest $argv; end
