#!/usr/bin/env bash
# pre-assert-hook for e2e-approve: runs on the test runner after the XR is
# applied and before up test waits for Ready. Plays the operator:
#   1. the gate holds: ApprovalRequired=False/WaitingForApproval, no ConfigMap;
#   2. approve via the status subresource, as documented in the README.
set -euo pipefail
NS=default
XR=e2e-approve

echo "Waiting for ${NS}/${XR} to wait for approval"
for _ in $(seq 1 60); do
  reason=$(kubectl get xapproval -n "$NS" "$XR" -o jsonpath='{.status.conditions[?(@.type=="ApprovalRequired")].reason}' 2>/dev/null || true)
  [ "$reason" = WaitingForApproval ] && break
  sleep 5
done
[ "$reason" = WaitingForApproval ] || { echo "FAIL: ApprovalRequired never became WaitingForApproval"; kubectl get xapproval -n "$NS" "$XR" -o yaml; exit 1; }

composed=$(kubectl get configmap -n "$NS" -l "crossplane.io/composite=${XR}" -o name)
[ -z "$composed" ] || { echo "FAIL: resources were composed before approval: $composed"; exit 1; }
echo "Gate holds: waiting for approval, nothing composed"

kubectl patch xapproval -n "$NS" "$XR" --subresource=status --type=merge -p '{"status":{"approved":true}}'
echo "Approved"
