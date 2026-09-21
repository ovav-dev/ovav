#!/bin/bash
# Commit canary artifacts to worktree
cd /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization

git add .ovav-ux-convergence/audit.sh \
        .ovav-ux-convergence/copy-audit.sh \
        .ovav-ux-convergence/backup.sh \
        .ovav-ux-convergence/ble-check-upstream.sh \
        .ovav-ux-convergence/ble-install-canary.sh \
        .ovav-ux-convergence/ble-validate-install.sh \
        .ovav-ux-convergence/blerc.template \
        .ovav-ux-convergence/ble-canary-test.sh \
        .ovav-ux-convergence/ble-interactive-canary.sh \
        .ovav-ux-convergence/ble-canary-fix.sh \
        .ovav-ux-convergence/ble-debug-bashrc.sh \
        .ovav-ux-convergence/ble-load-methods.sh \
        .ovav-ux-convergence/copy-debug.sh \
        .ovav-ux-convergence/script-canary.sh

git status --short | head -20
echo "---"
echo "Files staged. NOT yet committed (CEO approval required before permanent install)."