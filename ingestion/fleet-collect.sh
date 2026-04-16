#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026  Michael Spaeth
# Part of TrueState — https://gitea.local.vjinx.de/truestate-dev/truestate
#
# ingestion/fleet-collect.sh — collect packages from all hosts in an SSH fleet
# and register them as TrueState inventories.
#
# Reads Host aliases from an SSH config file, calls ingestion/collect.sh for
# each, and optionally triggers evaluations afterwards.
#
# Usage:
#   ./ingestion/fleet-collect.sh [OPTIONS]
#
#   --ssh-conf FILE    SSH config with Host entries [default: ~/.ssh/svc-automation.conf]
#   --hosts LIST       Comma-separated host aliases (default: all from SSH conf)
#   --api URL          TrueState API base URL       [default: http://localhost:8080]
#   --evaluate         Trigger evaluation for each collected inventory
#   --parallel N       Max concurrent collect jobs  [default: 4]
#   --dry-run          Print commands without executing
#
# The automation SSH key must be loaded before running:
#   eval $(~/bin/load-automation-cert)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COLLECT="${SCRIPT_DIR}/collect.sh"

SSH_CONF="${HOME}/.ssh/svc-automation.conf"
HOSTS_OVERRIDE=""
API_URL="http://localhost:8080"
DO_EVALUATE=0
PARALLEL=4
DRY_RUN=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --ssh-conf)  SSH_CONF="$2";         shift 2 ;;
        --hosts)     HOSTS_OVERRIDE="$2";   shift 2 ;;
        --api)       API_URL="$2";          shift 2 ;;
        --evaluate)  DO_EVALUATE=1;         shift   ;;
        --parallel)  PARALLEL="$2";        shift 2 ;;
        --dry-run)   DRY_RUN=1;            shift   ;;
        --help|-h)
            sed -n '2,/^set /p' "$0" | grep '^#' | sed 's/^# \?//'
            exit 0
            ;;
        *) echo "fleet-collect.sh: unknown option: $1" >&2; exit 2 ;;
    esac
done

if [[ ! -f "$COLLECT" ]]; then
    echo "fleet-collect.sh: collect.sh not found at ${COLLECT}" >&2
    exit 2
fi
if [[ ! -f "$SSH_CONF" ]]; then
    echo "fleet-collect.sh: SSH config not found: ${SSH_CONF}" >&2
    exit 2
fi

# ─── Parse host list ──────────────────────────────────────────────────────────
if [[ -n "$HOSTS_OVERRIDE" ]]; then
    IFS=',' read -ra HOSTS <<< "$HOSTS_OVERRIDE"
else
    # Extract Host aliases from SSH config: lines like "Host alias" (skip wildcards)
    mapfile -t HOSTS < <(
        grep -E '^[[:space:]]*Host[[:space:]]+[^*]+$' "$SSH_CONF" \
        | awk '{print $2}' \
        | sort -u
    )
fi

if [[ ${#HOSTS[@]} -eq 0 ]]; then
    echo "fleet-collect.sh: no hosts found in ${SSH_CONF}" >&2
    exit 2
fi

echo "[fleet] ${#HOSTS[@]} host(s): ${HOSTS[*]}" >&2
echo "[fleet] api=${API_URL} parallel=${PARALLEL} evaluate=${DO_EVALUATE}" >&2

# ─── Parallel collection ──────────────────────────────────────────────────────
RESULTS_DIR=$(mktemp -d)
trap 'rm -rf "${RESULTS_DIR}"' EXIT

pids=()
slots=()  # track running job count

run_collect() {
    local host="$1"
    local out="${RESULTS_DIR}/${host}.json"
    local log="${RESULTS_DIR}/${host}.log"

    if [[ "$DRY_RUN" -eq 1 ]]; then
        echo "[fleet] DRY RUN: collect.sh --host ${host} --api ${API_URL} --ssh-conf ${SSH_CONF}" >&2
        echo '{"id":"dry-run","name":"'"${host}"'"}' > "$out"
        return 0
    fi

    bash "$COLLECT" \
        --host "$host" \
        --api "$API_URL" \
        --ssh-conf "$SSH_CONF" \
        > "$out" 2>"$log" \
    && echo "[fleet] ✓ ${host}" >&2 \
    || { echo "[fleet] ✗ ${host} (see ${log})" >&2; rm -f "$out"; }
}

running=0
for host in "${HOSTS[@]}"; do
    # Throttle to PARALLEL concurrent jobs
    while [[ $running -ge $PARALLEL ]]; do
        wait -n 2>/dev/null || true
        running=$(jobs -r | wc -l)
    done

    run_collect "$host" &
    ((running++)) || true
done

# Wait for all remaining jobs
wait
echo "[fleet] collection complete" >&2

# ─── Gather inventory IDs ─────────────────────────────────────────────────────
declare -A INV_IDS  # host → inventory_id

for host in "${HOSTS[@]}"; do
    result="${RESULTS_DIR}/${host}.json"
    if [[ -f "$result" ]]; then
        inv_id=$(python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('id',''))" < "$result" 2>/dev/null || true)
        inv_name=$(python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('name',''))" < "$result" 2>/dev/null || true)
        if [[ -n "$inv_id" && "$inv_id" != "dry-run" ]]; then
            INV_IDS["$host"]="$inv_id"
            echo "[fleet] registered: ${host} → ${inv_name} (${inv_id})" >&2
        fi
    fi
done

echo "[fleet] ${#INV_IDS[@]} inventories registered" >&2

# ─── Optional: trigger evaluations ───────────────────────────────────────────
if [[ "$DO_EVALUATE" -eq 1 && ${#INV_IDS[@]} -gt 0 ]]; then
    echo "[fleet] triggering evaluations…" >&2
    for host in "${!INV_IDS[@]}"; do
        inv_id="${INV_IDS[$host]}"
        if [[ "$DRY_RUN" -eq 1 ]]; then
            echo "[fleet] DRY RUN: POST ${API_URL}/api/v1/evaluate/${inv_id}" >&2
            continue
        fi
        result=$(curl -sf -X POST "${API_URL}/api/v1/evaluate/${inv_id}" 2>/dev/null || true)
        if [[ -n "$result" ]]; then
            eval_id=$(echo "$result" | python3 -c "import json,sys; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || true)
            finding_count=$(echo "$result" | python3 -c "import json,sys; print(len(json.load(sys.stdin).get('findings',[])))" 2>/dev/null || true)
            echo "[fleet] eval: ${host} → ${finding_count} findings (eval ${eval_id})" >&2
        else
            echo "[fleet] eval failed: ${host} (${inv_id})" >&2
        fi
    done
    echo "[fleet] evaluations complete" >&2
fi

# ─── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "Summary:"
echo "  Hosts targeted:       ${#HOSTS[@]}"
echo "  Inventories created:  ${#INV_IDS[@]}"
echo "  Evaluations triggered: $([[ $DO_EVALUATE -eq 1 ]] && echo "${#INV_IDS[@]}" || echo "0 (use --evaluate)")"
