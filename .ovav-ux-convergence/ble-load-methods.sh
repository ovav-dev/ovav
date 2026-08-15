#!/bin/bash
# Try loading ble.sh in different ways to find one that works
WT=/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization
DEST="$WT/.ovav-ux-convergence"

# Method 1: Direct bash -i with --rcfile (no script wrapper)
cat > /tmp/opencode/method1-bashrc.sh <<'EOF'
export TERM=xterm-256color
echo "method1: BASH_SUBSHELL=$BASH_SUBSHELL" >&2
source ~/.local/share/blesh/ble.sh 2>&1 | head -3
echo "method1: BLE_VERSION=[$BLE_VERSION]" >&2
exit
EOF

# Method 2: source from inside an interactive bash that we exec into
cat > /tmp/opencode/method2-bashrc.sh <<'EOF'
export TERM=xterm-256color
echo "method2: BASH_SUBSHELL=$BASH_SUBSHELL" >&2
exec bash -i -c 'source ~/.local/share/blesh/ble.sh; echo "BLE_VERSION=$BLE_VERSION"; read -p "PAUSED" x'
EOF

# Method 3: use BASH_ENV (loaded for non-interactive shells but ble.sh needs i flag)
echo "Trying METHOD 1..."
RESULT=$(bash -i --rcfile /tmp/opencode/method1-bashrc.sh 2>&1)
echo "$RESULT"
echo "---"

echo "Trying METHOD 2 (exec bash)..."
# spawn a PTY using Python
python3 -c "
import pty, os, sys
pid, fd = pty.fork()
if pid == 0:
    os.execvp('bash', ['bash', '--rcfile', '/tmp/opencode/method2-bashrc.sh', '-i'])
else:
    import time
    output = b''
    start = time.time()
    while time.time() - start < 5:
        try:
            data = os.read(fd, 1024)
            output += data
        except OSError:
            break
    os.waitpid(pid, 0)
    print('METHOD 2 output:')
    print(output.decode('utf-8', errors='replace'))
" 2>&1 | head -40
echo "---"