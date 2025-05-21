package main

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
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

	// Should have no error results
	if len(rsp.GetResults()) > 0 {
		t.Errorf("expected no results but got: %v", rsp.GetResults())
	}

	// Should have an approval required condition
	if len(rsp.GetConditions()) == 0 {
		t.Fatal("expected at least one condition but got none")
	}

	// Check for the ApprovalRequired condition
	hasApprovalRequired := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == "ApprovalRequired" {
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

	// Should have no error results
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

	// Should have no error results
	if len(rsp.GetResults()) > 0 {
		t.Errorf("expected no results but got: %v", rsp.GetResults())
	}

	// Should have an approval required condition
	if len(rsp.GetConditions()) == 0 {
		t.Fatal("expected at least one condition but got none")
	}

	// Check for the ApprovalRequired condition
	hasApprovalRequired := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == "ApprovalRequired" {
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

	// Should have no error results
	if len(rsp.GetResults()) > 0 {
		t.Errorf("expected no results but got: %v", rsp.GetResults())
	}

	// Should have at least the ApprovalRequired condition
	if len(rsp.GetConditions()) < 1 {
		t.Fatalf("expected at least one condition but got: %d", len(rsp.GetConditions()))
	}

	// Should still have ApprovalRequired condition
	hasApprovalRequired := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == "ApprovalRequired" && cond.GetStatus() == fnv1.Status_STATUS_CONDITION_FALSE {
			hasApprovalRequired = true
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

func TestFunction_OverwriteDesiredWithObservedOnApprovalRequired(t *testing.T) {
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
							"test": "old-data"
						}
					},
					"status": {
						"approved": false,
						"oldHash": "` + oldHash + `"
					}
				}`),
			},
			Resources: map[string]*fnv1.Resource{
				"observed-resource": {
					Resource: resource.MustStructJSON(`{
						"apiVersion": "example.org/v1",
						"kind": "ObservedResource",
						"metadata": {
							"name": "observed-resource"
						},
						"spec": {
							"param": "old-value"
						}
					}`),
				},
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
							"test": "new-data" 
						}
					},
					"status": {
						"approved": false,
						"oldHash": "` + oldHash + `"
					}
				}`),
			},
			Resources: map[string]*fnv1.Resource{
				"desired-resource": {
					Resource: resource.MustStructJSON(`{
						"apiVersion": "example.org/v1",
						"kind": "DesiredResource",
						"metadata": {
							"name": "desired-resource"
						},
						"spec": {
							"param": "new-value"
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

	// Should have no error results
	if len(rsp.GetResults()) > 0 {
		t.Errorf("expected no results but got: %v", rsp.GetResults())
	}

	// Check that the desired composite resource was set to observed
	dxr := rsp.GetDesired().GetComposite()
	if dxr == nil {
		t.Fatal("expected desired composite resource but was nil")
	}

	// Check that the composite spec was overwritten with observed data
	resourceStruct := dxr.GetResource()
	if resourceStruct == nil {
		t.Fatal("resource struct is nil")
	}

	// Extract the "spec" field
	specField, ok := resourceStruct.GetFields()["spec"]
	if !ok {
		t.Fatal("spec field not found in resource")
	}

	// Get the resources field inside spec
	resourcesField, ok := specField.GetStructValue().GetFields()["resources"]
	if !ok {
		t.Fatal("resources field not found in spec")
	}

	// Get the test field inside resources
	testField, ok := resourcesField.GetStructValue().GetFields()["test"]
	if !ok {
		t.Fatal("test field not found in resources")
	}

	// Check if it has the old value
	if testField.GetStringValue() != "old-data" {
		t.Errorf("expected desired composite spec.resources.test to be overwritten with \"old-data\", got: %v",
			testField.GetStringValue())
	}

	// Check that the desired resources were overwritten with observed resources
	if len(rsp.GetDesired().GetResources()) != 1 {
		t.Errorf("expected 1 desired resource, got: %d", len(rsp.GetDesired().GetResources()))
	}

	_, hasObserved := rsp.GetDesired().GetResources()["observed-resource"]
	if !hasObserved {
		t.Error("expected desired resources to contain observed-resource but didn't")
	}

	_, hasDesired := rsp.GetDesired().GetResources()["desired-resource"]
	if hasDesired {
		t.Error("expected desired resources not to contain desired-resource but it did")
	}

	// Should have the ApprovalRequired condition
	hasApprovalRequired := false
	for _, cond := range rsp.GetConditions() {
		if cond.GetType() == "ApprovalRequired" {
			hasApprovalRequired = true
			break
		}
	}

	if !hasApprovalRequired {
		t.Error("expected to find ApprovalRequired condition but didn't")
	}
}
