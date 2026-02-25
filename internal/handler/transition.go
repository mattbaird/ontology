package handler

import "fmt"

// ValidateTransition checks whether transitioning from current to target is
// allowed according to the given transition map. It returns nil if the
// transition is valid, or a descriptive error otherwise.
func ValidateTransition(transitions map[string][]string, current, target string) error {
	allowed, ok := transitions[current]
	if !ok {
		return fmt.Errorf("unknown current state: %s", current)
	}
	for _, s := range allowed {
		if s == target {
			return nil
		}
	}
	return fmt.Errorf("transition from %q to %q is not allowed", current, target)
}
