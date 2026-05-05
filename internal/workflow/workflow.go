// Package workflow holds Temporal workflow definitions and activities.
//
// Phase 0: empty. Phase 1: ContractLifecycleWorkflow (intake -> review
// -> approve -> sign) and its activities (parse, extract, evaluate
// playbook, send for signature, ensure executed document).
//
// Workflows must be deterministic; all IO happens in activities.
package workflow
