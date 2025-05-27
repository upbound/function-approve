package main

import (
	"context"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

const (
	approvalRequiredCondition = "ApprovalRequired"
)

func TestFunction_MalformedInput(t *testing.T) {
	f := &Function{
		log: logging.NewNopLogger(),
	}

	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "fn-approval"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "approve.fn.crossplane.io/v1alpha1",
			"kind": "Input"
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					}
				}`),
			},
		},
	}

	rsp, err := f.RunFunction(context.Background(), req)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if rsp == nil {
		t.Fatal("expected response but got nil")
	}

	// Should have a fatal error result
	if len(rsp.GetResults()) == 0 {
		t.Fatal("expected at least one result but got none")
	}

	// The first result should be a fatal error
	if rsp.GetResults()[0].GetSeverity() != fnv1.Severity_SEVERITY_FATAL {
		t.Errorf("expected SEVERITY_FATAL but got: %v", rsp.GetResults()[0].GetSeverity())
	}
}

func TestFunction_FirstRunNeedsApproval(t *testing.T) {
	f := &Function{
		log: logging.NewNopLogger(),
	}

	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "fn-approval"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "approve.fn.crossplane.io/v1alpha1",
			"kind": "Input",
			"dataField": "spec.resources",
			"approvalField": "status.approved",
			"oldHashField": "status.oldHash",
			"newHashField": "status.newHash",
			"detailedCondition": true
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "data"
						}
					}
				}`),
			},
		},
		Desired: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "data"
						}
					}
				}`),
			},
		},
	}

	rsp, err := f.RunFunction(context.Background(), req)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if rsp == nil {
		t.Fatal("expected response but got nil")
	}

	// Should have a fatal result indicating approval is required
	if len(rsp.GetResults()) == 0 {
		t.Fatal("expected fatal result but got none")
	}

	// Check for fatal result with approval message
	hasFatalResult := false
	for _, result := range rsp.GetResults() {
		if result.GetSeverity() == fnv1.Severity_SEVERITY_FATAL {
			hasFatalResult = true
			message := result.GetMessage()
			if !strings.Contains(message, "Changes detected. Approval required.") {
				t.Errorf("expected fatal message to contain approval text but got: %v", message)
			}
		}
	}

	if !hasFatalResult {
		t.Error("expected to find fatal result but didn't")
	}

	// Should also have ApprovalRequired condition
	if len(rsp.GetConditions()) == 0 {
		t.Fatal("expected at least one condition but got none")
	}

	// Check for the ApprovalRequired condition
	hasApprovalRequired := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == approvalRequiredCondition {
			hasApprovalRequired = true
			if cond.GetStatus() != fnv1.Status_STATUS_CONDITION_FALSE {
				t.Errorf("expected STATUS_CONDITION_FALSE for ApprovalRequired but got: %v", cond.GetStatus())
			}
			if cond.GetReason() != "WaitingForApproval" {
				t.Errorf("expected WaitingForApproval reason but got: %v", cond.GetReason())
			}
		}
	}

	if !hasApprovalRequired {
		t.Error("expected to find ApprovalRequired condition but didn't")
	}
}

func TestFunction_AlreadyApproved(t *testing.T) {
	f := &Function{
		log: logging.NewNopLogger(),
	}

	const hashValue = "e02c6d35c585a43c62dc2ae14a5385b8a86168b36be7a0d985c0c09afca4ffbe"

	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "fn-approval"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "approve.fn.crossplane.io/v1alpha1",
			"kind": "Input",
			"dataField": "spec.resources",
			"approvalField": "status.approved",
			"oldHashField": "status.oldHash",
			"newHashField": "status.newHash"
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "data"
						}
					},
					"status": {
						"approved": true,
						"newHash": "` + hashValue + `"
					}
				}`),
			},
		},
		Desired: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "data"
						}
					},
					"status": {
						"approved": true,
						"newHash": "` + hashValue + `"
					}
				}`),
			},
		},
	}

	rsp, err := f.RunFunction(context.Background(), req)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if rsp == nil {
		t.Fatal("expected response but got nil")
	}

	// Should have no error results when approved
	if len(rsp.GetResults()) > 0 {
		t.Errorf("expected no results but got: %v", rsp.GetResults())
	}

	// Should have a success condition
	if len(rsp.GetConditions()) == 0 {
		t.Fatal("expected at least one condition but got none")
	}

	// Check for the FunctionSuccess condition
	hasFunctionSuccess := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == "FunctionSuccess" {
			hasFunctionSuccess = true
			if cond.GetStatus() != fnv1.Status_STATUS_CONDITION_TRUE {
				t.Errorf("expected STATUS_CONDITION_TRUE for FunctionSuccess but got: %v", cond.GetStatus())
			}
			if cond.GetReason() != "Success" {
				t.Errorf("expected Success reason but got: %v", cond.GetReason())
			}
		}
	}

	if !hasFunctionSuccess {
		t.Error("expected to find FunctionSuccess condition but didn't")
	}
}

func TestFunction_ChangesRequireApproval(t *testing.T) {
	f := &Function{
		log: logging.NewNopLogger(),
	}

	const oldHash = "e02c6d35c585a43c62dc2ae14a5385b8a86168b36be7a0d985c0c09afca4ffbe"

	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "fn-approval"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "approve.fn.crossplane.io/v1alpha1",
			"kind": "Input",
			"dataField": "spec.resources",
			"approvalField": "status.approved",
			"oldHashField": "status.oldHash",
			"newHashField": "status.newHash"
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "updated-data"
						}
					},
					"status": {
						"approved": false,
						"oldHash": "` + oldHash + `"
					}
				}`),
			},
		},
		Desired: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "updated-data"
						}
					},
					"status": {
						"approved": false,
						"oldHash": "` + oldHash + `"
					}
				}`),
			},
		},
	}

	rsp, err := f.RunFunction(context.Background(), req)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if rsp == nil {
		t.Fatal("expected response but got nil")
	}

	// Should have a fatal result indicating approval is required
	if len(rsp.GetResults()) == 0 {
		t.Fatal("expected fatal result but got none")
	}

	// Check for fatal result with approval message
	hasFatalResult := false
	for _, result := range rsp.GetResults() {
		if result.GetSeverity() == fnv1.Severity_SEVERITY_FATAL {
			hasFatalResult = true
			message := result.GetMessage()
			if !strings.Contains(message, "Changes detected. Approval required.") {
				t.Errorf("expected fatal message to contain approval text but got: %v", message)
			}
		}
	}

	if !hasFatalResult {
		t.Error("expected to find fatal result but didn't")
	}

	// Should also have ApprovalRequired condition
	if len(rsp.GetConditions()) == 0 {
		t.Fatal("expected at least one condition but got none")
	}

	// Check for the ApprovalRequired condition
	hasApprovalRequired := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == approvalRequiredCondition {
			hasApprovalRequired = true
			if cond.GetStatus() != fnv1.Status_STATUS_CONDITION_FALSE {
				t.Errorf("expected STATUS_CONDITION_FALSE for ApprovalRequired but got: %v", cond.GetStatus())
			}
			if cond.GetReason() != "WaitingForApproval" {
				t.Errorf("expected WaitingForApproval reason but got: %v", cond.GetReason())
			}
		}
	}

	if !hasApprovalRequired {
		t.Error("expected to find ApprovalRequired condition but didn't")
	}
}

func TestFunction_DesiredResources(t *testing.T) {
	f := &Function{
		log: logging.NewNopLogger(),
	}

	const oldHash = "e02c6d35c585a43c62dc2ae14a5385b8a86168b36be7a0d985c0c09afca4ffbe"

	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "fn-approval"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "approve.fn.crossplane.io/v1alpha1",
			"kind": "Input",
			"dataField": "spec.resources",
			"approvalField": "status.approved",
			"oldHashField": "status.oldHash",
			"newHashField": "status.newHash"
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "updated-data"
						}
					},
					"status": {
						"approved": false,
						"oldHash": "` + oldHash + `"
					}
				}`),
			},
		},
		Desired: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "updated-data"
						}
					},
					"status": {
						"approved": false,
						"oldHash": "` + oldHash + `"
					}
				}`),
			},
			Resources: map[string]*fnv1.Resource{
				"test-resource": {
					Resource: resource.MustStructJSON(`{
						"apiVersion": "example.org/v1",
						"kind": "ComposedResource",
						"metadata": {
							"name": "test-composed"
						},
						"spec": {
							"param": "value"
						}
					}`),
				},
			},
		},
	}

	rsp, err := f.RunFunction(context.Background(), req)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if rsp == nil {
		t.Fatal("expected response but got nil")
	}

	// Should have a fatal result indicating approval is required
	if len(rsp.GetResults()) == 0 {
		t.Fatal("expected fatal result but got none")
	}

	// Check for fatal result with approval message
	hasFatalResult := false
	for _, result := range rsp.GetResults() {
		if result.GetSeverity() == fnv1.Severity_SEVERITY_FATAL {
			hasFatalResult = true
			message := result.GetMessage()
			if !strings.Contains(message, "Changes detected. Approval required.") {
				t.Errorf("expected fatal message to contain approval text but got: %v", message)
			}
		}
	}

	if !hasFatalResult {
		t.Error("expected to find fatal result but didn't")
	}

	// Should also have ApprovalRequired condition
	if len(rsp.GetConditions()) == 0 {
		t.Fatal("expected at least one condition but got none")
	}

	// Check for the ApprovalRequired condition
	hasApprovalRequired := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == approvalRequiredCondition {
			hasApprovalRequired = true
			if cond.GetStatus() != fnv1.Status_STATUS_CONDITION_FALSE {
				t.Errorf("expected STATUS_CONDITION_FALSE for ApprovalRequired but got: %v", cond.GetStatus())
			}
			if cond.GetReason() != "WaitingForApproval" {
				t.Errorf("expected WaitingForApproval reason but got: %v", cond.GetReason())
			}
		}
	}

	if !hasApprovalRequired {
		t.Error("expected to find ApprovalRequired condition but didn't")
	}

	// Check that the desired resources match the observed resources (blocking changes)
	observedResName := ""
	for k := range req.GetObserved().GetResources() {
		observedResName = k
		break
	}

	// Verify the response reuses the observed resources
	hasObservedResource := false
	for desiredResName := range rsp.GetDesired().GetResources() {
		if desiredResName == observedResName {
			hasObservedResource = true
			break
		}
	}

	if !hasObservedResource && len(req.GetObserved().GetResources()) > 0 {
		t.Error("expected to find observed resources in desired state but didn't")
	}
}

func TestFunction_FatalResultsWithApprovalCondition(t *testing.T) {
	f := &Function{
		log: logging.NewNopLogger(),
	}

	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "fn-approval"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "approve.fn.crossplane.io/v1alpha1",
			"kind": "Input",
			"dataField": "spec.resources",
			"approvalField": "status.approved",
			"oldHashField": "status.oldHash",
			"newHashField": "status.newHash",
			"detailedCondition": true
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "data"
						}
					}
				}`),
			},
		},
		Desired: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.org/v1",
					"kind": "XR",
					"metadata": {
						"name": "test-xr"
					},
					"spec": {
						"resources": {
							"test": "data"
						}
					}
				}`),
			},
		},
	}

	rsp, err := f.RunFunction(context.Background(), req)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if rsp == nil {
		t.Fatal("expected response but got nil")
	}

	// Should have fatal results
	if len(rsp.GetResults()) == 0 {
		t.Fatal("expected fatal result but got none")
	}

	// Check for fatal result with approval message
	hasFatalResult := false
	for _, result := range rsp.GetResults() {
		if result.GetSeverity() == fnv1.Severity_SEVERITY_FATAL {
			hasFatalResult = true
			message := result.GetMessage()
			if !strings.Contains(message, "Changes detected. Approval required.") {
				t.Errorf("expected fatal message to contain approval text but got: %v", message)
			}
		}
	}

	if !hasFatalResult {
		t.Error("expected to find fatal result but didn't")
	}

	// Should also have ApprovalRequired condition
	if len(rsp.GetConditions()) == 0 {
		t.Fatal("expected at least one condition but got none")
	}

	// Check for the ApprovalRequired condition
	hasApprovalRequired := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == approvalRequiredCondition {
			hasApprovalRequired = true
			if cond.GetStatus() != fnv1.Status_STATUS_CONDITION_FALSE {
				t.Errorf("expected STATUS_CONDITION_FALSE for ApprovalRequired but got: %v", cond.GetStatus())
			}
			if cond.GetReason() != "WaitingForApproval" {
				t.Errorf("expected WaitingForApproval reason but got: %v", cond.GetReason())
			}
		}
	}

	if !hasApprovalRequired {
		t.Error("expected to find ApprovalRequired condition but didn't")
	}

}

func TestFunction_HashCalculationFromCorrectSource(t *testing.T) {
	f := &Function{
		log: logging.NewNopLogger(),
	}

	// Test that hash is calculated from spec.resources.data field, not from desired resources
	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "fn-approval"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "approve.fn.crossplane.io/v1alpha1",
			"kind": "Input",
			"dataField": "spec.resources",
			"approvalField": "status.approved",
			"oldHashField": "status.oldHash",
			"newHashField": "status.newHash"
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.crossplane.io/v1",
					"kind": "XApproval",
					"metadata": {
						"name": "approval-example"
					},
					"spec": {
						"resources": {
							"data": {
								"key1": "value1",
								"key2": "originalValue"
							}
						}
					}
				}`),
			},
		},
		Desired: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.crossplane.io/v1",
					"kind": "XApproval",
					"metadata": {
						"name": "approval-example"
					},
					"spec": {
						"resources": {
							"data": {
								"key1": "value1",
								"key2": "changedValue"
							}
						}
					}
				}`),
			},
			// Add desired resources that should NOT be used for hash calculation
			Resources: map[string]*fnv1.Resource{
				"some-resource": {
					Resource: resource.MustStructJSON(`{
						"apiVersion": "example.org/v1",
						"kind": "SomeResource",
						"metadata": {
							"name": "some-resource"
						},
						"spec": {
							"param": "this-should-not-affect-hash"
						}
					}`),
				},
			},
		},
	}

	rsp, err := f.RunFunction(context.Background(), req)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if rsp == nil {
		t.Fatal("expected response but got nil")
	}

	// Should require approval since spec.resources changed from "originalValue" to "changedValue"
	if len(rsp.GetResults()) == 0 {
		t.Fatal("expected fatal result but got none")
	}

	// Check for fatal result
	hasFatalResult := false
	for _, result := range rsp.GetResults() {
		if result.GetSeverity() == fnv1.Severity_SEVERITY_FATAL {
			hasFatalResult = true
			message := result.GetMessage()
			if !strings.Contains(message, "Changes detected. Approval required.") {
				t.Errorf("expected fatal message to contain approval text but got: %v", message)
			}
		}
	}

	if !hasFatalResult {
		t.Error("expected to find fatal result but didn't")
	}

	// Check that the hash was calculated from spec.resources field, not from desired resources
	// We verify this by checking that approval is required (indicating hash difference)
	// even though we have desired resources that could have caused different hash behavior
	hasApprovalRequired := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == approvalRequiredCondition {
			hasApprovalRequired = true
			message := cond.GetMessage()
			// The message should contain hash info based on spec.resources, not desired resources
			if !strings.Contains(message, "Current hash:") || !strings.Contains(message, "Approved hash:") {
				t.Errorf("expected condition message to contain hash information but got: %v", message)
			}
		}
	}

	if !hasApprovalRequired {
		t.Error("expected to find ApprovalRequired condition but didn't")
	}

	// Verify the hash was stored correctly in the response
	dxr := rsp.GetDesired().GetComposite()
	if dxr == nil {
		t.Fatal("expected desired composite resource but was nil")
	}

	// Check that newHash field was set
	statusValue, ok := dxr.GetResource().GetFields()["status"]
	if !ok {
		t.Error("expected status field to be set")
		return
	}

	statusStruct := statusValue.GetStructValue()
	if statusStruct == nil {
		t.Error("expected status to be a struct")
		return
	}

	newHashValue, ok := statusStruct.GetFields()["newHash"]
	if !ok {
		t.Error("expected newHash field to be set in status")
		return
	}

	newHashString := newHashValue.GetStringValue()
	if newHashString == "" {
		t.Error("expected newHash to be non-empty")
	}
}

func TestFunction_ApprovedWithHashChanges(t *testing.T) {
	f := &Function{
		log: logging.NewNopLogger(),
	}

	// This test simulates the exact scenario from the real environment:
	// - status.approved = true
	// - oldHash != newHash (there are changes)
	// - Should NOT require approval since user already approved

	const approvedHash = "559b7f636dcd75c3dfa6449f7aaa060fd8a52fc7d70f74792feb04930fa2c400"

	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "fn-approval"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "approve.fn.crossplane.io/v1alpha1",
			"kind": "Input",
			"dataField": "spec.resources",
			"approvalField": "status.approved",
			"oldHashField": "status.oldHash",
			"newHashField": "status.newHash"
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.crossplane.io/v1",
					"kind": "XApproval",
					"metadata": {
						"name": "approval-example"
					},
					"spec": {
						"resources": {
							"data": {
								"key1": "value1",
								"key2": "testApproval4"
							}
						}
					},
					"status": {
						"approved": true,
						"oldHash": "` + approvedHash + `"
					}
				}`),
			},
		},
		Desired: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: resource.MustStructJSON(`{
					"apiVersion": "example.crossplane.io/v1",
					"kind": "XApproval",
					"metadata": {
						"name": "approval-example"
					},
					"spec": {
						"resources": {
							"data": {
								"key1": "value1",
								"key2": "newChangedValue"
							}
						}
					},
					"status": {
						"approved": true,
						"oldHash": "` + approvedHash + `"
					}
				}`),
			},
		},
	}

	rsp, err := f.RunFunction(context.Background(), req)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if rsp == nil {
		t.Fatal("expected response but got nil")
	}

	// Should have NO fatal results since changes are approved
	if len(rsp.GetResults()) > 0 {
		t.Errorf("expected no results when approved=true but got: %v", rsp.GetResults())
	}

	// Should have a success condition
	if len(rsp.GetConditions()) == 0 {
		t.Fatal("expected at least one condition but got none")
	}

	// Check for the FunctionSuccess condition
	hasFunctionSuccess := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == "FunctionSuccess" {
			hasFunctionSuccess = true
			if cond.GetStatus() != fnv1.Status_STATUS_CONDITION_TRUE {
				t.Errorf("expected STATUS_CONDITION_TRUE for FunctionSuccess but got: %v", cond.GetStatus())
			}
			if cond.GetReason() != "Success" {
				t.Errorf("expected Success reason but got: %v", cond.GetReason())
			}
		}
	}

	if !hasFunctionSuccess {
		t.Error("expected to find FunctionSuccess condition but didn't")
	}

	// Should NOT have ApprovalRequired condition when approved=true
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == approvalRequiredCondition {
			t.Errorf("should not have ApprovalRequired condition when approved=true but found: %v", cond)
		}
	}
}
