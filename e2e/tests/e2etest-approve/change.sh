#!/usr/bin/env bash
# post-assert-hook for e2e-approve: runs after the XR is Ready. Plays a user
# changing an approved XR, then the operator approving the change:
#   1. change spec.resources: the gate closes again and the ConfigMap keeps the
#      approved data;
#   2. approve: the change goes through.
set -euo pipefail
NS=default
XR=e2e-approve

cm_version() {
  kubectl get configmap -n "$NS" -l "crossplane.io/composite=${XR}" -o jsonpath='{.items[0].data.version}'
}

approved_hash=$(kubectl get xapproval -n "$NS" "$XR" -o jsonpath='{.status.currentHash}')
[ -n "$approved_hash" ] || { echo "FAIL: no approved hash in status.currentHash"; exit 1; }
[ "$(cm_version)" = v1 ] || { echo "FAIL: ConfigMap version is '$(cm_version)', want v1"; exit 1; }

kubectl patch xapproval -n "$NS" "$XR" --type=merge -p '{"spec":{"resources":{"data":{"version":"v2"}}}}'

# The new request names the hash approved in round one; the first round's
# condition named none.
echo "Waiting for ${NS}/${XR} to wait for approval of the change"
for _ in $(seq 1 60); do
  msg=$(kubectl get xapproval -n "$NS" "$XR" -o jsonpath='{.status.conditions[?(@.type=="ApprovalRequired")].message}')
  case "$msg" in *"Approved hash: ${approved_hash}"*) break ;; esac
  sleep 5
done
case "$msg" in *"Approved hash: ${approved_hash}"*) ;; *) echo "FAIL: no approval request for the change; condition: $msg"; exit 1 ;; esac

[ "$(cm_version)" = v1 ] || { echo "FAIL: unapproved change reached the ConfigMap: version $(cm_version)"; exit 1; }
echo "Gate holds: change waits for approval, ConfigMap still v1"

kubectl patch xapproval -n "$NS" "$XR" --subresource=status --type=merge -p '{"status":{"approved":true}}'
echo "Approved; waiting for the change to reach the ConfigMap"
for _ in $(seq 1 60); do
  [ "$(cm_version)" = v2 ] && { echo "ConfigMap updated to v2"; exit 0; }
  sleep 5
done
echo "FAIL: approved change never reached the ConfigMap: version $(cm_version)"
exit 1
