#!/bin/bash
# Copy external audit output into workspace (script is in workspace, allowed)
set -e
DEST=/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/AUDIT-1.txt
cp /tmp/opencode/ovav-ux-audit.txt "$DEST"
ls -la "$DEST"