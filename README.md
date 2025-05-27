# function-approve

A Crossplane Composition Function for implementing manual approval workflows.

## Overview

The `function-approve` provides a serverless approval mechanism at the Crossplane level that:

1. Tracks changes to a specified field by computing a hash
2. When changes need approval, uses fatal results to halt pipeline execution
3. Requires explicit approval before allowing changes to proceed
4. Prevents drift by storing previously approved state

This function implements the approval workflow using Crossplane's fatal result mechanism, ensuring that no changes are applied until explicitly approved.

## Usage

Add the function to your Crossplane installation:

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: function-approve
spec:
  package: xpkg.upbound.io/upbound/function-approve:v0.1.0
```

### How It Works

1. When a resource is created or updated, `function-approve` calculates a hash of the monitored field (e.g., `spec.resources`).
2. The function compares this hash with the previously approved hash stored in `status.currentHash`.
3. If the hashes don't match and approval is not granted, the function returns a fatal result to halt pipeline execution.
4. An operator must approve the change by setting `status.approved = true`.
5. After approval, the new hash is stored as `currentHash`, the approval flag is reset, and changes are allowed to proceed.
6. If a customer modifies an existing claim after approval, this will generate a new hash, requiring another approval.

## Example

```yaml
apiVersion: example.crossplane.io/v1
kind: Composition
metadata:
  name: approval-workflow-example
spec:
  compositeTypeRef:
    apiVersion: example.crossplane.io/v1
    kind: XR
  pipeline:
  - step: require-approval
    functionRef:
      name: function-approve
    input:
      apiVersion: approve.fn.crossplane.io/v1alpha1
      kind: Input
      dataField: "spec.resources"  # Field to monitor for changes
      approvalField: "status.approved"
      currentHashField: "status.currentHash"
      detailedCondition: true
```

## Input Configuration Options

| Field | Type | Description |
|-------|------|-------------|
| `dataField` | string | **Required**. Field to monitor for changes (e.g., `spec.resources`) |
| `approvalField` | string | Status field to check for approval. Default: `status.approved` |
| `currentHashField` | string | Status field to store the approved hash. Default: `status.currentHash` |
| `detailedCondition` | bool | Whether to add detailed information to conditions. Default: `true` |
| `approvalMessage` | string | Message to display when approval is required. Default: `Changes detected. Approval required.` |

## Using with Custom Resources

Your XR definition must include the status fields used by the function:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xapprovals.example.crossplane.io
spec:
  group: example.crossplane.io
  names:
    kind: XApproval
    plural: xapprovals
  versions:
  - name: v1
    served: true
    referenceable: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              resources:
                type: object
                x-kubernetes-preserve-unknown-fields: true
          status:
            type: object
            properties:
              approved:
                type: boolean
                description: "Whether the current changes are approved"
              currentHash:
                type: string
                description: "Hash of the currently approved resource state"
```

## Approving Changes

When changes are detected, the function returns a fatal result (halting pipeline execution) and the resource will show an `ApprovalRequired` condition. To approve the changes, patch the resource's status:

```yaml
kubectl patch xapproval example --type=merge --subresource=status -p '{"status":{"approved":true}}'
```

After approval, the function will:
1. Update `currentHash` to the new approved hash
2. Reset the approval flag to `false`
3. Allow the pipeline to continue normally

## Resetting Approval State

If you need to reset the approval state, you can clear the `currentHash` field:

```yaml
kubectl patch xapproval example --type=merge --subresource=status -p '{"status":{"currentHash":""}}'
```

## Security Considerations

- Use RBAC to control who can approve changes by restricting access to the status subresource
- Consider implementing additional verification steps or multi-party approval in your workflow

## How Changes Are Prevented

The function uses fatal results to prevent changes when approval is required:

1. When changes are detected but not yet approved, the function:
   - Returns a fatal result to halt pipeline execution
   - Sets an ApprovalRequired condition for visibility
   - Provides detailed information about the required approval

2. This approach has several benefits:
   - Deterministic behavior - pipeline execution is completely stopped
   - Clear error messaging to operators about required approvals
   - Works consistently across different Crossplane versions
   - Clean separation between approval logic and resource state

## Complete Example

Here's a complete example of a composition using `function-approve`:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: approval-required-cluster
spec:
  compositeTypeRef:
    apiVersion: example.crossplane.io/v1
    kind: XCluster
  pipeline:
  - step: require-approval
    functionRef:
      name: function-approve
    input:
      apiVersion: approve.fn.crossplane.io/v1alpha1
      kind: Input
      dataField: "spec.resources"
      approvalField: "status.approved"
      currentHashField: "status.currentHash"
      detailedCondition: true
      approvalMessage: "Cluster changes require admin approval"
  - step: create-resources
    functionRef:
      name: function-patch-and-transform
    input:
      apiVersion: pt.fn.crossplane.io/v1alpha1
      kind: Resources
      resources:
      - name: cluster
        base:
          apiVersion: eks.aws.upbound.io/v1beta1
          kind: Cluster
          spec:
            forProvider:
              region: us-west-2
              version: "1.25"
```

## Metrics and Monitoring

- Monitor resources with the `ApprovalRequired` condition to track pending approvals
- Implement alerting based on condition status for timely approvals
- Consider tracking approval times and frequencies to optimize your workflows
