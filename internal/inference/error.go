package inference

import "fmt"

type StatusError struct {
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("inference server returned status %d", e.StatusCode)
}
