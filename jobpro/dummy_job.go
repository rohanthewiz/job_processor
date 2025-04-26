package jobpro

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// DummyJob is a job that simulates work by sleeping
type DummyJob struct {
	BaseJob
	successProb float64 // Probability of success (0.0-1.0)
}

// NewDummyJob creates a new dummy job
func NewDummyJob(id, name string, schedType FreqType,
	workTime time.Duration, successProb float64) *DummyJob {

	job := &DummyJob{
		BaseJob: BaseJob{
			id:          id,
			name:        name,
			freqType:    schedType,
			maxWorkTime: workTime,
		},
		successProb: successProb,
	}

	// Set the work function
	job.BaseJob.workFunc = job.dummyWork

	return job
}

// dummyWork is the work function for DummyJob
func (j *DummyJob) dummyWork(ctx context.Context) (string, error) {
	// Simulate some work by sleeping
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(j.maxWorkTime):
		// Determine success based on probability
		if rand.Float64() < j.successProb {
			return fmt.Sprintf("Job %s completed successfully", j.id), nil
		}
		return "", fmt.Errorf("job %s failed with simulated error", j.id)
	}
}
