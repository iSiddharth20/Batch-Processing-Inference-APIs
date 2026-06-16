package inference

import "context"

// Client is the seam to the rate-limited inference server. Tests inject a fake.
type Client interface {
	Infer(ctx context.Context, prompt string) (string, error)
}
