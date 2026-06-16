package storage

import (
	"errors"

	"github.com/iSiddharth20/Batch-Processing-Inference-APIs/internal/domain"
)

var ErrJobNotFound = errors.New("job not found")

// JobView is a race-safe snapshot of a job for status reads.
type JobView struct {
	ID        string
	Status    domain.Status
	Total     int
	Completed int
	Results   []domain.Result
	Errors    []domain.PromptError
}

type Store interface {
	CreateJob(job *domain.Job) error
	GetJob(id string) (JobView, error)
	RecordResult(id string, result domain.Result) error
	RecordError(id string, promptErr domain.PromptError) error
	MarkDone(id string) error
}
