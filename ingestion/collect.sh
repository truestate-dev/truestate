#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026  Michael Spaeth
# Part of TrueState — https://gitea.local.vjinx.de/truestate-dev/truestate
#
# ingestion/collect.sh — collect installed packages from a host and register
# (or update) it as a TrueState inventory.
#
# Reads all installed packages via dpkg-query, detects platform/release from
# /etc/os-release, and POSTs to the TrueState API.
#
# Usage:
#   ./ingestion/collect.sh [OPTIONS]
#
#   --host    HOST       SSH target (alias in SSH config, or user@ip)      [required]
#   --name    NAME       Inventory display name (default: HOST)
#   --type    TYPE       Inventory type: host | golden (default: host)
#   --api     URL        TrueState API base URL (default: http://localhost:8080)
#   --ssh-conf FILE      SSH config file (default: ~/.ssh/svc-automation.conf)
#   --dry-run            Print the JSON payload without POSTing
#
# The automation SSH key must be loaded before running (unless already in agent):
#   eval $(~/bin/load-automation-cert)

set -euo pipefail

# ─── Defaults ────────────────────────────────────────────────────────────────
HOST=""
INV_NAME=""
INV_TYPE="host"
API_URL="http://localhost:8080"
SSH_CONF="${HOME}/.ssh/svc-automation.conf"
DRY_RUN=0

# ─── Argument parsing ─────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --host)      HOST="$2";      shift 2 ;;
        --name)      INV_NAME="$2";  shift 2 ;;
        --type)      INV_TYPE="$2";  shift 2 ;;
        --api)       API_URL="$2";   shift 2 ;;
        --ssh-conf)  SSH_CONF="$2";  shift 2 ;;
        --dry-run)   DRY_RUN=1;      shift   ;;
        --help|-h)
            sed -n '2,/^set /p' "$0" | grep '^#' | sed 's/^# \?//'
            exit 0
            ;;
        *) echo "collect.sh: unknown option: $1" >&2; exit 2 ;;
    esac
done

if [[ -z "$HOST" ]]; then
    echo "collect.sh: --host is required" >&2
    exit 2
fi

case "$INV_TYPE" in
    host|golden) ;;
    *) echo "collect.sh: --type must be 'host' or 'golden'" >&2; exit 2 ;;
esac

INV_NAME="${INV_NAME:-$HOST}"

# ─── SSH helper ───────────────────────────────────────────────────────────────
_SSH_OPTS=(-o StrictHostKeyChecking=no -o BatchMode=yes -o ConnectTimeout=10)
if [[ -f "$SSH_CONF" ]]; then
    _SSH_OPTS+=(-F "$SSH_CONF")
fi
rem() { ssh "${_SSH_OPTS[@]}" "$HOST" "$@" </dev/null; }

# ─── Collect OS info ─────────────────────────────────────────────────────────
echo "[collect] SSH → ${HOST}: reading /etc/os-release" >&2
os_raw=$(rem "cat /etc/os-release 2>/dev/null" | tr -d '\r')

platform=$(python3 -c "
import sys
d = {}
for line in sys.stdin:
    k, _, v = line.strip().partition('=')
    d[k] = v.strip().strip('\"')
id_val = d.get('ID', '').lower()
if 'ubuntu' in id_val:
    print('ubuntu')
elif 'proxmox' in id_val or 'pve' in id_val:
    print('proxmox')
else:
    print('debian')
" <<< "$os_raw")

release=$(python3 -c "
import sys, re
d = {}
for line in sys.stdin:
    k, _, v = line.strip().partition('=')
    d[k] = v.strip().strip('\"')
codename = d.get('VERSION_CODENAME', '').lower()
if codename:
    print(codename)
else:
    version = d.get('VERSION', '')
    m = re.search(r'\(([^)]+)\)', version)
    if m:
        print(m.group(1).lower())
    else:
        print(d.get('VERSION_ID', 'unknown').lower())
" <<< "$os_raw")

echo "[collect] platform=${platform} release=${release}" >&2

# ─── Collect packages via dpkg-query ─────────────────────────────────────────
echo "[collect] collecting packages (dpkg-query)" >&2
pkg_tsv=$(rem "dpkg-query -W -f='\${Package}\t\${Version}\t\${Architecture}\n' 2>/dev/null" | tr -d '\r')

pkg_count=$(echo "$pkg_tsv" | grep -c . || true)
echo "[collect] ${pkg_count} packages found" >&2

pkg_json=$(python3 -c "
import json, sys
packages = []
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    parts = line.split('\t')
    if len(parts) < 2:
        continue
    name    = parts[0]
    version = parts[1] if len(parts) > 1 else ''
    arch    = parts[2] if len(parts) > 2 else ''
    if name and version:
        packages.append({'name': name, 'version': version, 'arch': arch})
print(json.dumps(packages))
" <<< "$pkg_tsv")

# ─── Build inventory payload ──────────────────────────────────────────────────
payload=$(python3 -c "
import json, sys
packages = json.loads(sys.argv[4])
payload = {
    'name':     sys.argv[1],
    'type':     sys.argv[2],
    'platform': sys.argv[3],
    'release':  sys.argv[5],
    'packages': packages,
    'metadata': {'collector': 'collect.sh', 'source_host': sys.argv[6]},
}
print(json.dumps(payload))
" "$INV_NAME" "$INV_TYPE" "$platform" "$pkg_json" "$release" "$HOST")

# ─── Post to API (or dry-run) ─────────────────────────────────────────────────
if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "[collect] DRY RUN — payload (truncated to first 2 packages):" >&2
    echo "$payload" | python3 -c "
import json, sys
d = json.load(sys.stdin)
d['packages'] = d['packages'][:2]
d['_note'] = '... truncated for display'
print(json.dumps(d, indent=2))
"
    exit 0
fi

echo "[collect] POST ${API_URL}/api/v1/inventories" >&2
response=$(curl -sf \
    -X POST \
    -H "Content-Type: application/json" \
    -d "$payload" \
    "${API_URL}/api/v1/inventories")

inv_id=$(echo "$response"   | python3 -c "import json,sys; print(json.load(sys.stdin).get('id',''))")
inv_name=$(echo "$response" | python3 -c "import json,sys; print(json.load(sys.stdin).get('name',''))")

echo "[collect] registered inventory: id=${inv_id} name=${inv_name}" >&2
echo "$response"
