#!/usr/bin/env bash
# Rewrites every datapoint's `at` in the collect payloads so the newest point
# is "now" and the series spans the last ~6 hours. Run before the demo so
# charts and @at columns show live-looking data.
set -euo pipefail

cd "$(dirname "$0")" >/dev/null

python3 - <<'EOF'
import json, glob, time

now_ms = int(time.time() * 1000)

for path in glob.glob("multisensor-*.json"):
    with open(path) as f:
        payload = json.load(f)

    # Collect all timestamps to compute the global span
    all_ts = sorted(
        dp["at"]
        for ds in payload["datastreams"]
        for dp in ds["datapoints"]
        if "at" in dp
    )
    if not all_ts:
        continue
    newest = all_ts[-1]
    shift = now_ms - newest

    for ds in payload["datastreams"]:
        for dp in ds["datapoints"]:
            if "at" in dp:
                dp["at"] += shift

    with open(path, "w") as f:
        json.dump(payload, f, indent=2)
    print(f"{path}: shifted {len(all_ts)} datapoints (+{shift} ms)")
EOF

echo "Done. Newest datapoint is now ~current time."
