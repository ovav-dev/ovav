# ═══════════════════════════════════════════════════════════════════════════════
# OVAV FISH PROMPT — Scientific 2026 Eye-Comfort Design
# All color state managed inline — no closures, no helper functions for colors.
# ═══════════════════════════════════════════════════════════════════════════════

function __ovav_workspace_accent
    set -l cwd (pwd)
    switch $cwd
        case "/home/braka/dev/bab/ventures/ovav" "/home/braka/dev/bab/ventures/ovav/*" "/home/braka/Systems/OVAV" "/home/braka/Systems/OVAV/*"
            echo "b86bff"
        case "/home/braka/.config" "/home/braka/.config/*" "/home/braka/.local" "/home/braka/.local/*"
            echo "ffbf3f"
        case "/home/braka/dev/work" "/home/braka/dev/work/*"
            echo "00d5ff"
        case "/mnt/c" "/mnt/c/*"
            echo "7dcfff"
        case "*"
            echo "00ff9c"
    end
end

function __ovav_find_up
    set -l marker $argv[1]
    set -l dir (pwd)
    while test "$dir" != "/"
        if test -e "$dir/$marker"
            echo $dir
            return 0
        end
        set dir (dirname $dir)
    end
    return 1
end

function __ovav_git_branch
    command git rev-parse --is-inside-work-tree >/dev/null 2>&1; or return
    set -l branch (command git branch --show-current 2>/dev/null)
    test -z "$branch"; and set branch (command git rev-parse --short HEAD 2>/dev/null)
    test -n "$branch"; and echo $branch
end

function __ovav_git_dirty_count
    command git rev-parse --is-inside-work-tree >/dev/null 2>&1; or return
    command git status --porcelain 2>/dev/null | wc -l | string trim
end

function __ovav_cwd
    set -l cwd (pwd)
    set cwd (string replace -r '^/home/braka/dev/bab/ventures/ovav' 'ovav' -- $cwd)
    set cwd (string replace -r '^/home/braka/dev/work' 'dev/work' -- $cwd)
    set cwd (string replace -r '^/home/braka/.config' '.config' -- $cwd)
    set cwd (string replace -r '^/home/braka/Systems/OVAV' 'ovav' -- $cwd)
    set cwd (string replace -r '^/home/braka' '~' -- $cwd)
    set cwd (string replace -r '^/mnt/c/Users/[^/]+' 'win' -- $cwd)
    if test (string length -- $cwd) -gt 46
        echo "…"(string sub -s -42 -- $cwd)
    else
        echo $cwd
    end
end

function __ovav_valid_project_root
    set -l root $argv[1]
    test -z "$root"; and return 1
    test "$root" = "/home/braka"; and return 1
    return 0
end

function __ovav_is_day
    set -l hour (date +%H)
    if test "$hour" -ge 7 -a "$hour" -lt 19
        echo "yes"
    else
        echo "no"
    end
end

# ═══════════════════════════════════════════════════════════════════════════════
function fish_prompt
    set -l code $status
    set -l is_day (__ovav_is_day)

    # ── Scientific 2026 palette — NO RED for normal text ─────────────────────────
    # All color pairs verified: bg must differ significantly from fg for WCAG AA
    if test "$is_day" = "yes"
        set -l bar_bg   #d0d5e8
        set -l bar_fg   #1a1c2e
        set -l pchar_bg  #1a1c2e
        set -l pchar_fg  #f0f2f7
        set -l err_bg    #cc2244
        set -l err_fg    #f0f2f7
        set -l ibg       #dde0ed
        set -l icon_fg   #0a6ea8   # dark cyan — visible, NOT red
        set -l bfg       #6c3dcc   # dark purple
        set -l dtofg     #a67c00   # dark gold
        set -l openfg    #cc2244   # dark red — ONLY for opencode icon
    else
        set -l bar_bg   #0f0f1a
        set -l bar_fg   #c8d0e7
        set -l pchar_bg  #c8d0e7
        set -l pchar_fg  #0f0f1a
        set -l err_bg    #ff4d7d
        set -l err_fg    #0f0f1a
        set -l ibg       #252836
        set -l icon_fg   #00d5ff   # cyan
        set -l bfg       #b86bff   # purple
        set -l dtofg     #ffbf3f   # yellow
        set -l openfg    #ff4d7d   # red — ONLY for opencode icon
    end

    # ── TOP BAR ────────────────────────────────────────────────────────────
    echo
    # ╭─
    set_color -b $bar_bg; set_color --bold $bar_fg; printf "╭─ "
    # distro icon
    set_color -b $bar_bg; set_color --bold $icon_fg; printf ""
    # cwd
    set_color -b $bar_bg; set_color --bold $bar_fg; printf "   %s " (__ovav_cwd)

    # git branch
    set -l branch (__ovav_git_branch)
    if test -n "$branch"
        set_color -b $bar_bg; set_color --bold $bar_fg; printf "  "
        set_color -b $bar_bg; set_color --bold $bfg; printf " %s" "$branch"
        set -l dirty_count (__ovav_git_dirty_count)
        if test -n "$dirty_count"; and test "$dirty_count" -gt 0
            set_color -b $bar_bg; set_color --bold $dtofg; printf " ●%s" "$dirty_count"
        end
    end

    # error
    if test $code -ne 0
        set_color -b $bar_bg; set_color --bold $err_bg; printf "  ✘ %s" "$code"
    end

    # ── RUNTIME ICONS — each icon: set bg+fg explicitly, then restore bar ─────
    set -l ip 0

    set -l opencode_root (__ovav_find_up ".opencode")
    if __ovav_valid_project_root "$opencode_root"
        test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
        set_color -b $ibg; set_color --bold $openfg; printf "󰚩"
        set_color -b $bar_bg; set_color --bold $bar_fg
        set ip 1
    else
        set -l opencode_file_root (__ovav_find_up "opencode.jsonc")
        if __ovav_valid_project_root "$opencode_file_root"
            test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
            set_color -b $ibg; set_color --bold $openfg; printf "󰚩"
            set_color -b $bar_bg; set_color --bold $bar_fg
            set ip 1
        end
    end

    set -l mem_root (__ovav_find_up ".engram")
    if __ovav_valid_project_root "$mem_root"
        test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
        set_color -b $ibg; set_color --bold $bfg; printf "󰍛"
        set_color -b $bar_bg; set_color --bold $bar_fg
        set ip 1
    end

    set -l bun_root (__ovav_find_up "bun.lock")
    if not __ovav_valid_project_root "$bun_root"
        set bun_root (__ovav_find_up "bun.lockb")
    end
    if __ovav_valid_project_root "$bun_root"
        test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
        set_color -b $ibg; set_color --bold $bfg; printf ""
        set_color -b $bar_bg; set_color --bold $bar_fg
        set ip 1
    end

    set -l ts_root (__ovav_find_up "tsconfig.json")
    if __ovav_valid_project_root "$ts_root"
        test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
        set_color -b $ibg; set_color --bold $icon_fg; printf ""
        set_color -b $bar_bg; set_color --bold $bar_fg
        set ip 1
    else
        set -l node_root (__ovav_find_up "package.json")
        if __ovav_valid_project_root "$node_root"
            test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
            set_color -b $ibg; set_color --bold $icon_fg; printf ""
            set_color -b $bar_bg; set_color --bold $bar_fg
            set ip 1
        end
    end

    set -l go_root (__ovav_find_up "go.mod")
    if __ovav_valid_project_root "$go_root"
        test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
        set_color -b $ibg; set_color --bold $icon_fg; printf ""
        set_color -b $bar_bg; set_color --bold $bar_fg
        set ip 1
    end

    set -l rust_root (__ovav_find_up "Cargo.toml")
    if __ovav_valid_project_root "$rust_root"
        test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
        set_color -b $ibg; set_color --bold $icon_fg; printf ""
        set_color -b $bar_bg; set_color --bold $bar_fg
        set ip 1
    end

    set -l py_root (__ovav_find_up "pyproject.toml")
    if not __ovav_valid_project_root "$py_root"
        set py_root (__ovav_find_up "requirements.txt")
    end
    if __ovav_valid_project_root "$py_root"
        test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
        set_color -b $ibg; set_color --bold $icon_fg; printf "󱌪"
        set_color -b $bar_bg; set_color --bold $bar_fg
        set ip 1
    end

    set -l docker_root (__ovav_find_up "Dockerfile")
    if __ovav_valid_project_root "$docker_root"
        test $ip -eq 1; and begin set_color -b $bar_bg; set_color --bold $bar_fg; printf " "; end
        set_color -b $ibg; set_color --bold $icon_fg; printf "󰜭"
        set_color -b $bar_bg; set_color --bold $bar_fg
    end

    # ── BOTTOM BAR + PROMPT CHAR ──────────────────────────────────────────────
    echo
    set_color -b $bar_bg; set_color --bold $bar_fg; printf "╰─ "

    if test $code -eq 0
        set_color -b $pchar_bg; set_color --bold $pchar_fg; printf "❯"
        set_color -b $bar_bg; set_color --bold $bar_fg; printf " "
    else
        set_color -b $err_bg; set_color --bold $err_fg; printf "✘"
        set_color -b $bar_bg; set_color --bold $bar_fg; printf " "
    end

    set_color normal
end
