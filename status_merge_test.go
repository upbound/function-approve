package main

import (
	"context"
	"testing"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

// An operator approved the change (observed status.approved=true) and an
// earlier pipeline step wrote an unrelated status field to the desired XR. The
// approval must still count, the hash must be saved and the earlier step's
// status field must be kept.
func TestFunction_ApprovalWithStatusFromEarlierStep(t *testing.T) {
	f := &Function{log: logging.NewNopLogger()}
	req := &fnv1.RunFunctionRequest{
		Input: resource.MustStructJSON(`{"apiVersion":"approve.fn.crossplane.io/v1alpha1","kind":"Input","dataField":"spec.resources"}`),
		Observed: &fnv1.State{Composite: &fnv1.Resource{Resource: resource.MustStructJSON(
			`{"apiVersion":"example.crossplane.io/v1","kind":"XR","metadata":{"name":"x"},"spec":{"resources":{"a":1}},"status":{"approved":true}}`)}},
		Desired: &fnv1.State{Composite: &fnv1.Resource{Resource: resource.MustStructJSON(
			`{"apiVersion":"example.crossplane.io/v1","kind":"XR","metadata":{"name":"x"},"spec":{"resources":{"a":1}},"status":{"endpoint":"from-earlier-step"}}`)}},
	}

	rsp, err := f.RunFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("RunFunction: %v", err)
	}
	for _, r := range rsp.GetResults() {
		if r.GetSeverity() == fnv1.Severity_SEVERITY_FATAL {
			t.Fatalf("approved change was halted: %s", r.GetMessage())
		}
	}

	status := rsp.GetDesired().GetComposite().GetResource().GetFields()["status"].GetStructValue().GetFields()
	if status["currentHash"].GetStringValue() == "" {
		t.Errorf("currentHash not saved: %v", status)
	}
	if status["approved"].GetBoolValue() {
		t.Errorf("approved not reset to false: %v", status)
	}
	if got := status["endpoint"].GetStringValue(); got != "from-earlier-step" {
		t.Errorf("earlier step's status field = %q, want from-earlier-step", got)
	}
}

func TestMergeStatus(t *testing.T) {
	dst := map[string]interface{}{"approved": true, "nested": map[string]interface{}{"a": "observed", "b": "observed"}}
	mergeStatus(dst, map[string]interface{}{"endpoint": "x", "nested": map[string]interface{}{"b": "desired"}})
	nested := dst["nested"].(map[string]interface{})
	if dst["approved"] != true || dst["endpoint"] != "x" || nested["a"] != "observed" || nested["b"] != "desired" {
		t.Errorf("merged = %v", dst)
	}
}
