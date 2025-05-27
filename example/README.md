# Approval Function Examples

This directory contains examples demonstrating the function-approve capabilities for implementing approval workflows in Crossplane.

## Prerequisites

To run these examples, you need:

1. The Crossplane CLI installed
2. function-approve built and available

## Understanding the Approval Function

The approval function provides a controlled way to manage changes to Crossplane resources, requiring manual approval before changes are applied. This is useful for scenarios where you want to review changes before they are applied to your infrastructure.

The function works by:
1. Computing a hash of specified data in your composite resource
2. Checking if the hash has changed since the last approved version
3. Pausing reconciliation if changes are detected and approval is needed
4. Resuming reconciliation once changes are approved

## Running the Examples

### Running the Example

Render the example to see how the approval function works:

```shell
crossplane render xr.yaml composition.yaml functions.yaml
```

The function will detect changes and require approval before allowing the pipeline to continue.

## Configuration Options

The function supports these configuration options:

- `dataField`: Specifies which field to monitor for changes (required)
- `approvalField`: Status field to check for approval (default: "status.approved")
- `currentHashField`: Where to store the approved hash (default: "status.currentHash")
- `detailedCondition`: Whether to include hash details in conditions (default: true)
- `approvalMessage`: Custom message for approval required condition

## Approval Workflow

1. Make changes to the resource's spec
2. The function detects changes and halts the pipeline execution
3. Review the changes through the resource's conditions and fatal results
4. Set the approval field (default: `status.approved: true`)
5. The function detects approval and allows the pipeline to continue
6. The `currentHash` is updated to reflect the newly approved state

## Example Configuration

Here's a typical function configuration:

```yaml
apiVersion: approve.fn.crossplane.io/v1alpha1
kind: Input
dataField: "spec.resources"
approvalField: "status.approved"
currentHashField: "status.currentHash"
detailedCondition: true
approvalMessage: "Changes detected. Administrative approval required."
```

The function uses a fatal result approach to halt pipeline execution when approval is required,
ensuring that no changes are applied until explicitly approved.
