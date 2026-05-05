package workflow

// Workflow identity + version constants. Bump WorkflowVersion when you
// patch a workflow definition; in-flight executions need workflow.GetVersion
// shims to bridge old and new logic.
const (
	ContractLifecycleWorkflowName = "pactline.ContractLifecycleWorkflow"
	WorkflowTaskQueue             = "pactline"
	ContractApprovalSignalName    = "approval_decision"
)

func WorkflowID(contractID string) string { return "contract:" + contractID }
