package inference

import (
	"context"
	"math/rand/v2"
	"net/http"
	"time"
)

type MockServer struct {
	rateLimitChance float64
	latency         time.Duration
}

func NewMockServer(rateLimitChance float64, latency time.Duration) *MockServer {
	return &MockServer{rateLimitChance: rateLimitChance, latency: latency}
}

func (m *MockServer) Infer(ctx context.Context, prompt string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(m.latency):
	}

	if rand.Float64() < m.rateLimitChance {
		return "", &StatusError{StatusCode: http.StatusTooManyRequests}
	}
	return "mock inference response", nil
}
