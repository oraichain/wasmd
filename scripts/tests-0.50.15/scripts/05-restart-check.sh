#!/usr/bin/env bash
# After the v0.50.15 upgrade: stop and cold-restart all 3 nodes on the NEW binary.
#
# This is the real test of declaring StoreUpgrades.Deleted WITHOUT unmounting the
# keys in app.go: the multistore re-runs loadVersion on every process start, and a
# store that is still mounted but no longer in CommitInfo triggers
#   "version of store <name> mismatch root store's version"
# So the chain can look fine right after the upgrade and then fail to reboot.
set -euo pipefail
source "$(dirname "$0")/lib.sh"

compose ps --status running 2>/dev/null | grep -q tests-0.50.15-node1 \
  || die "network not running — run 03 + 04 first"

H_BEFORE=$(height_of node1)
log "cold restart node1/2/3 on ${IMAGE_NEW} (height before ${H_BEFORE})"
export NODE_IMAGE="${IMAGE_NEW}"
compose stop
compose up -d --force-recreate

FAILED=0
if wait_height node1 "$(( ${H_BEFORE:-0} + 2 ))" 60; then
  ok "node1 resumed after cold restart (height $(height_of node1))"
else
  warn "node1 did NOT resume — checking logs"
  compose logs --tail 40 node1 2>&1 | grep -iE 'mismatch|store|panic|error|UPGRADE' | tail -20 || true
  FAILED=1
fi
for n in node2 node3; do
  wait_height "${n}" "$(( ${H_BEFORE:-0} + 1 ))" 40 || { warn "${n} did not resume"; FAILED=1; }
done

echo
if [[ ${FAILED} -eq 0 ]]; then
  printf '\033[1;32mPASS\033[0m  all 3 nodes cold-restart cleanly on the NEW binary after the store deletion.\n'
else
  printf '\033[1;31mFAIL\033[0m  a cold restart broke — StoreUpgrades.Deleted needs the matching app.go unmount.\n'
  compose logs --tail 15 node1 2>&1 | tail -15 || true
fi
exit ${FAILED}
