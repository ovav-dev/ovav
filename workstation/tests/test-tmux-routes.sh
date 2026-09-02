#!/usr/bin/env bash
# End-to-end tmux route test: raw Alt bytes → tmux key table → named window.
set -euo pipefail

OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/ovav}"
AKRYNT_ROOT="${AKRYNT_ROOT:-/home/braka/Systems/projects/work/akrynt-agent}"

test -d "$OVAV_ROOT"
test -d "$AKRYNT_ROOT"
test -f "$HOME/.tmux.conf"

OVAV_ROOT="$OVAV_ROOT" AKRYNT_ROOT="$AKRYNT_ROOT" python3 - <<'PY'
import os
import pty
import signal
import subprocess
import time

socket = f"ovav-alt-route-{os.getpid()}"
ovav_root = os.environ["OVAV_ROOT"]
akrynt_root = os.environ["AKRYNT_ROOT"]


def tmux(*args: str) -> str:
    result = subprocess.run(
        ["tmux", "-L", socket, *args],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


proc = None
master = None
slave = None
try:
    tmux("-f", os.path.expanduser("~/.tmux.conf"), "new-session", "-d", "-s", "test", "-n", "home", "-c", os.path.expanduser("~"))
    tmux("new-window", "-d", "-t", "test:", "-n", "ovav", "-c", ovav_root)
    tmux("new-window", "-d", "-t", "test:", "-n", "akrynt", "-c", akrynt_root)

    master, slave = pty.openpty()
    env = os.environ.copy()
    env["TERM"] = "xterm-256color"
    proc = subprocess.Popen(
        ["tmux", "-L", socket, "attach-session", "-t", "test"],
        stdin=slave,
        stdout=slave,
        stderr=slave,
        env=env,
        start_new_session=True,
    )
    os.close(slave)
    slave = None
    time.sleep(0.5)

    results = []
    for raw, expected in ((b"\x1b1", "home"), (b"\x1b2", "ovav"), (b"\x1b3", "akrynt")):
        os.write(master, raw)
        time.sleep(0.2)
        actual = tmux("display-message", "-p", "#{window_name}")
        results.append((raw.hex(), actual, expected))

    for raw, actual, expected in results:
        if actual != expected:
            raise SystemExit(f"FAIL: {raw} routed to {actual!r}, expected {expected!r}")
        print(f"PASS: {raw} → {actual}")
finally:
    if proc is not None and proc.poll() is None:
        proc.send_signal(signal.SIGHUP)
        try:
            proc.wait(timeout=2)
        except subprocess.TimeoutExpired:
            proc.kill()
    if master is not None:
        os.close(master)
    if slave is not None:
        os.close(slave)
    subprocess.run(["tmux", "-L", socket, "kill-server"], capture_output=True)
PY
