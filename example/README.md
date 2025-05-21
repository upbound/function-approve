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

### Basic Example with Pause Annotation

The default behavior uses the `crossplane.io/paused` annotation to pause reconciliation:

```shell
crossplane render xr.yaml composition.yaml functions.yaml
```

### Example with Synced=False Condition

This alternative approach uses the Synced=False condition instead of the pause annotation:

```shell
crossplane render xr.yaml composition.yaml functions.yaml
```

The `setSyncedFalse: true` option in the composition enables this behavior.

## Configuration Options

The function supports these configuration options:

- `dataField`: Specifies which field to monitor for changes (required)
- `approvalField`: Status field to check for approval (default: "status.approved")
- `oldHashField`: Where to store the approved hash (default: "status.oldHash")
- `newHashField`: Where to store the current hash (default: "status.newHash")
- `pauseAnnotation`: Annotation used to pause reconciliation (default: "crossplane.io/paused")
- `detailedCondition`: Whether to include hash details in conditions (default: true)
- `approvalMessage`: Custom message for approval required condition
- `setSyncedFalse`: Use Synced=False condition instead of pause annotation (default: false)

## Approval Workflow

1. Make changes to the resource's spec
2. The function detects changes and pauses reconciliation
3. Review the changes through the resource's conditions
4. Set the approval field (default: `status.approved: true`)
5. The function detects approval and resumes reconciliation

## Example With setSyncedFalse

In some environments, it may be preferable to use Synced=False condition instead of annotations.
The `setSyncedFalse: true` option enables this alternative approach:

```yaml
apiVersion: approve.fn.crossplane.io/v1alpha1
kind: Input
dataField: "spec.resources"
approvalField: "status.approved"
newHashField: "status.newHash"
oldHashField: "status.oldHash"
detailedCondition: true
approvalMessage: "Changes detected. Administrative approval required."
setSyncedFalse: true
```

When enabled, this sets the Synced condition to False instead of adding the pause annotation,
providing the same pause functionality through a different mechanism.
