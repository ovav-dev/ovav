#!/usr/bin/env bash
# add_memory_dir_allow.sh
# Adds /home/braka/Labs/mimocode/data/memory/* to external_directory allow rules
# in both runtime configs. Idempotent.
#
# Run as Braka (CEO) or with sudo. No waiver required — this is a config edit
# under runtime_governance scope (ovav-memory-bridge skill).
#
# After running, RESTART the agent session to reload the permission snapshot.

set -euo pipefail

OVAV_ROOT="/home/braka/Systems/OVAV"
MIMOCODE_GLOBAL="${OVAV_ROOT}/.mimocode/global_config/config.json"
OPENCODE_CONFIG="${OVAV_ROOT}/opencode.json"

MEMORY_PATH="/home/braka/Labs/mimocode/data/memory/*"

# 1. Patch .mimocode/global_config/config.json
if ! grep -qF "${MEMORY_PATH}" "${MIMOCODE_GLOBAL}"; then
    echo "[+] Adding ${MEMORY_PATH} to ${MIMOCODE_GLOBAL}"
    python3 -c "
import json, sys
p = '${MIMOCODE_GLOBAL}'
with open(p) as f:
    cfg = json.load(f)
ed = cfg['permission'].setdefault('external_directory', {})
ed['${MEMORY_PATH}'] = 'allow'
with open(p, 'w') as f:
    json.dump(cfg, f, indent=2)
print('OK')
"
else
    echo "[=] ${MEMORY_PATH} already in ${MIMOCODE_GLOBAL}"
fi

# 2. Patch opencode.json
if ! grep -qF "${MEMORY_PATH}" "${OPENCODE_CONFIG}"; then
    echo "[+] Adding ${MEMORY_PATH} to ${OPENCODE_CONFIG}"
    python3 -c "
import json, sys
p = '${OPENCODE_CONFIG}'
with open(p) as f:
    cfg = json.load(f)
ed = cfg.get('permission', {}).setdefault('external_directory', {})
ed['${MEMORY_PATH}'] = 'allow'
with open(p, 'w') as f:
    json.dump(cfg, f, indent=2)
print('OK')
"
else
    echo "[=] ${MEMORY_PATH} already in ${OPENCODE_CONFIG}"
fi

# 3. Patch .ovav/source/opencode/config.yaml (so OVAV Product installs include the rule)
OVAV_SOURCE="${OVAV_ROOT}/.ovav/source/opencode/config.yaml"
if [[ -f "${OVAV_SOURCE}" ]] && ! grep -qF "${MEMORY_PATH}" "${OVAV_SOURCE}"; then
    echo "[+] Adding ${MEMORY_PATH} to ${OVAV_SOURCE}"
    python3 -c "
import yaml, sys
p = '${OVAV_SOURCE}'
with open(p) as f:
    cfg = yaml.safe_load(f)
ed = cfg.get('permission', {}).setdefault('external_directory', {})
ed['${MEMORY_PATH}'] = 'allow'
with open(p, 'w') as f:
    yaml.safe_dump(cfg, f, default_flow_style=False, sort_keys=False)
print('OK')
"
else
    echo "[=] ${MEMORY_PATH} already in ${OVAV_SOURCE} or file not found"
fi

echo
echo "DONE. Restart agent session for changes to take effect."
echo "Verify with: grep '${MEMORY_PATH}' ${MIMOCODE_GLOBAL} ${OPENCODE_CONFIG}"