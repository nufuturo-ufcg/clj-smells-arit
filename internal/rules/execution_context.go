package rules

// ExecutionContext describes when an AST node is known to execute. Rules that
// report load-time behavior must require ExecutionAtLoad: deferred, quoted, or
// unknown macro bodies are deliberately not evidence of load-time execution.
type ExecutionContext string

const (
	ExecutionAtLoad       ExecutionContext = "load"
	ExecutionDeferred     ExecutionContext = "deferred"
	ExecutionNonEvaluated ExecutionContext = "non-evaluated"
	ExecutionUnknown      ExecutionContext = "unknown"
)

func CurrentExecutionContext(context map[string]interface{}) ExecutionContext {
	if execution, ok := context["executionContext"].(ExecutionContext); ok {
		return execution
	}
	return ExecutionUnknown
}

func ExecutesAtLoad(context map[string]interface{}) bool {
	return CurrentExecutionContext(context) == ExecutionAtLoad
}
