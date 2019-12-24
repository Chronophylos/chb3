package actions

import "fmt"

type noPermissionError struct {
	has    Permission
	needed Permission
}

func (e *noPermissionError) Error() string {
	return fmt.Sprintf(
		"Needed Permission is not high enough (has: %d, needed: %d)",
		e.has, e.needed,
	)
}
