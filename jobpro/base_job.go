package jobpro

import (
	"context"
	"fmt"
	"time"
)

// BaseJob is a basic implementation of the Job interface
// To use, compose it in to a higher level job
type BaseJob struct {
	id          string
	name        string
	freqType    FreqType
	workFunc    func(context.Context) (string, error)
	maxWorkTime time.Duration
}

/*// NewBaseJob creates a new BaseJob with the given parameters
func NewBaseJob(id, name string, freqType FreqType, workTime time.Duration,
	workFunc func(context.Context) (string, error)) *BaseJob {
	return &BaseJob{
		id:          id,
		name:        name,
		freqType:    freqType,
		maxWorkTime: workTime,
		workFunc:    workFunc,
	}
}
*/

// Run executes the job's workFunc and returns stats
func (j *BaseJob) Run(ctx context.Context) (Stats, error) {
	stats := Stats{
		StartTimeUTC: time.Now().UTC(),
	}

	const maxTimeGracePeriod = 5 * time.Second

	// Create a timer for the job duration
	var timer *time.Timer
	if j.maxWorkTime > 0 {
		timer = time.NewTimer(j.maxWorkTime)
		defer timer.Stop()
	}

	type Result struct {
		msg string
		err error
	}

	// Create a channel for the job result
	resultCh := make(chan Result, 1)

	// Run the actual worker
	go func() {
		msg, err := j.workFunc(ctx) // Run it!
		resultCh <- Result{msg, err}
	}()

	// Wait for either the job to complete, the timer to expire, or the context to be canceled
	select {
	case <-ctx.Done():
		// Job was canceled
		stats.Duration = time.Since(stats.StartTimeUTC)
		stats.SuccessMsg = "Job was canceled"
		return stats, ctx.Err()

	case <-j.timerOrNil(timer):
		// Job duration exceeded, but still wait for result
		select {
		case result := <-resultCh:
			stats.Duration = time.Since(stats.StartTimeUTC)
			stats.SuccessMsg = result.msg
			return stats, result.err
		case <-time.After(maxTimeGracePeriod): // Small grace period
			stats.Duration = time.Since(stats.StartTimeUTC)
			stats.SuccessMsg = "Job timed out but didn't respect cancellation"
			return stats, fmt.Errorf("job execution exceeded maximum duration")
		}

	case result := <-resultCh:
		// Job completed successfully
		stats.Duration = time.Since(stats.StartTimeUTC)
		stats.SuccessMsg = result.msg
		return stats, result.err
	}
}

// timerOrNil takes a timer and returns the timer's channel or nil
// This is just a convenience function for use in a select statement to conditionally
// timeout a process or not
// Note that a nil channel will never be "selected"
func (j *BaseJob) timerOrNil(timer *time.Timer) <-chan time.Time {
	if timer == nil {
		return nil
	}
	return timer.C
}

// ID returns the job ID
func (j *BaseJob) ID() string {
	return j.id
}

// Name returns the job name
func (j *BaseJob) Name() string {
	return j.name
}

// Type returns the job schedule type
func (j *BaseJob) Type() FreqType {
	return j.freqType
}
