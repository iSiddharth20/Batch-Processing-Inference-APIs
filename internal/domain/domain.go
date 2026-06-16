package domain

import (
	"sync/atomic"
)

// Status is the lifecycle state of a batch job.
type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

// Prompt is a single unit of work within a batch.
type Prompt struct {
	ID         int // unique within the batch
	PromptText string
	RetryCount int
}

// Result is a successful inference for one prompt.
type Result struct {
	ID                int // matches the originating Prompt.ID
	InferenceResponse string
}

// PromptError is a per-prompt failure.
type PromptError struct {
	ID           int // matches the originating Prompt.ID
	StatusCode   int
	ErrorMessage string
}

// Job is the record for one batch,
// persists for the life of the job.
type Job struct {
	ID               string
	Status           Status
	TotalPrompts     int
	CompletedPrompts atomic.Int64
	RetryCount       int
	Prompts          []Prompt
	Results          []Result
	PerPromptErrors  []PromptError
}
