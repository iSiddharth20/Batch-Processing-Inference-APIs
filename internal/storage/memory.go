package storage

import (
	"fmt"
	"sync"

	"github.com/iSiddharth20/Batch-Processing-Inference-APIs/internal/domain"
)

type MemoryStore struct {
	mu   sync.Mutex
	jobs map[string]*domain.Job
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{jobs: make(map[string]*domain.Job)}
}

func (s *MemoryStore) CreateJob(job *domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job %s already exists", job.ID)
	}
	s.jobs[job.ID] = job
	return nil
}

func (s *MemoryStore) GetJob(id string) (JobView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return JobView{}, ErrJobNotFound
	}
	return JobView{
		ID:        job.ID,
		Status:    job.Status,
		Total:     job.TotalPrompts,
		Completed: int(job.CompletedPrompts.Load()),
		Results:   job.Results,
		Errors:    job.PerPromptErrors,
	}, nil
}

func (s *MemoryStore) RecordResult(id string, result domain.Result) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return ErrJobNotFound
	}
	job.Results = append(job.Results, result)
	job.CompletedPrompts.Add(1)
	return nil
}

func (s *MemoryStore) RecordError(id string, promptErr domain.PromptError) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return ErrJobNotFound
	}
	job.PerPromptErrors = append(job.PerPromptErrors, promptErr)
	job.CompletedPrompts.Add(1)
	return nil
}

func (s *MemoryStore) MarkDone(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return ErrJobNotFound
	}
	job.Status = domain.StatusDone
	return nil
}
