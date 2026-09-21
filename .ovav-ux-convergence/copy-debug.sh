#!/bin/bash
# Copy debug log into workspace
cp /tmp/opencode/canary-rcfile-debug.log /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/DEBUG-LOG.txt
echo "Copied debug log to workspace"
wc -l /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/DEBUG-LOG.txt