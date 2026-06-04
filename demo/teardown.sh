#!/usr/bin/env bash
# Tears down EVERYTHING the demo created, so the demo can be run again from
# scratch. Tolerant: every step is skipped cleanly if the artifact is gone.
#
# Usage: demo/teardown.sh [--keep-provision]
#   --keep-provision   keep the datamodel and the 3 devices (faster re-demo;
#                      skip steps 1-2 of the runbook on the next run)
set -uo pipefail

cd "$(dirname "$0")/.." >/dev/null
OG=${OG:-./og}
ORG=${OG_DEMO_ORG:-sensehat}
KEEP_PROVISION=false
[ "${1:-}" = "--keep-provision" ] && KEEP_PROVISION=true

step() { printf '\n\033[1m== %s\033[0m\n' "$*"; }

step "Workspace + dashboard"
$OG workspace delete _multisensor_demo_ws 2>/dev/null \
  && echo "  workspace deleted" || echo "  (no workspace, skipped)"

step "Close demo alarms"
$OG alarms search -w "alarm.entityIdentifier like multisensor" -o json 2>/dev/null \
  | python3 -c "
import json, sys
try:
    alarms = json.load(sys.stdin)
except Exception:
    alarms = []
for a in alarms:
    if a.get('status') != 'CLOSED' and a.get('identifier'):
        print(a['identifier'])
" | while read -r id; do
    $OG alarms close "$id" --notes "demo teardown" >/dev/null 2>&1 \
      && echo "  closed alarm $id"
  done
echo "  (alarm history keeps CLOSED entries — that's platform behavior)"

step "Rules"
$OG rules search -o json 2>/dev/null | python3 -c "
import json, sys
try:
    rules = json.load(sys.stdin)
except Exception:
    rules = []
for r in rules:
    if r.get('name') in ('Battery low', 'Environmental anomaly'):
        print(r['identifier'])
" | while read -r id; do
    $OG rules delete "$id" --org "$ORG" >/dev/null 2>&1 && echo "  deleted rule $id"
  done

step "Operation type"
$OG optypes delete CALIBRATE_SENSOR --org "$ORG" 2>/dev/null \
  && echo "  CALIBRATE_SENSOR deleted" || echo "  (no optype, skipped)"
echo "  (launched jobs stay in history — platform keeps the execution record)"

if [ "$KEEP_PROVISION" = true ]; then
  step "Provision kept (--keep-provision)"
else
  step "Devices"
  for n in 1 2 3; do
    $OG dev delete "multisensor-00$n" --org "$ORG" 2>/dev/null \
      && echo "  multisensor-00$n deleted" || echo "  (multisensor-00$n absent, skipped)"
  done

  step "Datamodel"
  $OG dm delete multisensor --org "$ORG" 2>/dev/null \
    && echo "  multisensor datamodel deleted" || echo "  (no datamodel, skipped)"
fi

step "Done"
echo "Sandbox clean. Re-run the demo with demo/setup.sh or step by step (demo/README.md)."
