#!/usr/bin/env bash
# Runs the whole demo unattended (rehearsal / smoke test). For a live
# presentation, run the steps manually from demo/README.md instead.
set -euo pipefail

cd "$(dirname "$0")/.." >/dev/null
OG=${OG:-./og}
ORG=${OG_DEMO_ORG:-sensehat}

step() { printf '\n\033[1m== %s\033[0m\n' "$*"; }

step "1. Datamodel"
$OG dm create --org "$ORG" -f demo/datamodels/multisensor.json 2>/dev/null \
  && echo "  created" || echo "  (already exists, skipped)"

step "2. Devices"
for n in 1 2 3; do
  $OG dev create --org "$ORG" -f "demo/devices/multisensor-00$n.json" 2>/dev/null \
    && echo "  multisensor-00$n created" || echo "  (multisensor-00$n already exists, skipped)"
done

step "3. Telemetry"
demo/collect/refresh-timestamps.sh
for n in 1 2 3; do
  $OG iot collect-file "multisensor-00$n" -f "demo/collect/multisensor-00$n.json"
done
$OG dev search -w "provision.device.identifier like multisensor" \
  -s provision.device.identifier -s sensor.temperature@at -s power.battery@at

step "4. Rules (EASY + ADVANCED)"
$OG rules deploy demo/rules/default_channel/battery-low --org "$ORG"
$OG rules deploy demo/rules/default_channel/env-anomaly --org "$ORG"
$OG rules search -w "rule.active eq true"

step "5. Trigger alarms"
$OG iot collect multisensor-001 power.battery 12
$OG iot collect multisensor-002 sensor.humidity 30
sleep 2
$OG iot collect multisensor-002 sensor.temperature 31
sleep 8
$OG alarms search -w "alarm.entityIdentifier like multisensor"

step "6. Custom operation"
$OG optypes create --org "$ORG" -f demo/operations/types/calibrate-sensor.json 2>/dev/null \
  && echo "  CALIBRATE_SENSOR created" || echo "  (optype already exists, skipped)"
sleep 2
$OG jobs create -f demo/operations/jobs/calibrate-001.json | head -4

step "7. Workspace + dashboard"
$OG workspace deploy demo/workspaces/multisensor-demo

if [ -n "${OG_DEMO_SHARE_WITH:-}" ]; then
  step "8. Share with $OG_DEMO_SHARE_WITH"
  $OG workspace share _multisensor_demo_ws --user "$OG_DEMO_SHARE_WITH"
fi

step "Demo ready"
echo "Open the OpenGate web UI → workspace 'Multisensor Demo' → 'Multisensor Overview'."
echo "Share it:           og workspace share _multisensor_demo_ws --user <email>"
echo "Undo everything with demo/teardown.sh"
