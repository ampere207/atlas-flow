package runtime

import "errors"

var (
	ErrInvalidDefinition = errors.New("invalid workflow definition")
	ErrDuplicateTaskID   = errors.New("duplicate task id in workflow definition")
	ErrUnknownDependency = errors.New("workflow definition contains an unknown dependency")
	ErrCyclicDefinition  = errors.New("workflow definition contains a cycle")
)
