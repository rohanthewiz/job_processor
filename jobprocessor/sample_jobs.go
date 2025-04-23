package jobprocessor

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// SimpleJob is a basic implementation of the Job interface
type SimpleJob struct {
	id        string
	name      string
	schedType ScheduleType
	workFunc  func(context.Context) (string, error)
	workTime  time.Duration
}

// NewSimpleJob creates a new simple job with the given parameters
func NewSimpleJob(id, name string, schedType ScheduleType, workTime time.Duration,
	workFunc func(context.Context) (string, error)) *SimpleJob {
	return &SimpleJob{
		id:        id,
		name:      name,
		schedType: schedType,
		workTime:  workTime,
		workFunc:  workFunc,
	}
}

// Run executes the job and returns stats
func (j *SimpleJob) Run(ctx context.Context) (Stats, error) {
	stats := Stats{
		StartTimeUTC: time.Now().UTC(),
	}

	// Create a timer for the job duration
	timer := time.NewTimer(j.workTime)
	defer timer.Stop()

	// Create a channel for the job result
	resultCh := make(chan struct {
		msg string
		err error
	}, 1)

	// Run the job in a goroutine
	go func() {
		msg, err := j.workFunc(ctx)
		resultCh <- struct {
			msg string
			err error
		}{msg, err}
	}()

	// Wait for either the job to complete, the timer to expire, or the context to be canceled
	select {
	case <-ctx.Done():
		// Job was canceled
		stats.Duration = time.Since(stats.StartTimeUTC)
		stats.SuccessMsg = "Job was canceled"
		return stats, ctx.Err()
	case <-timer.C:
		// Job duration exceeded, but still wait for result
		select {
		case result := <-resultCh:
			stats.Duration = time.Since(stats.StartTimeUTC)
			stats.SuccessMsg = result.msg
			return stats, result.err
		case <-time.After(500 * time.Millisecond): // Small grace period
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

// ID returns the job ID
func (j *SimpleJob) ID() string {
	return j.id
}

// Name returns the job name
func (j *SimpleJob) Name() string {
	return j.name
}

// Type returns the job schedule type
func (j *SimpleJob) Type() ScheduleType {
	return j.schedType
}

// DummyJob is a job that simulates work by sleeping
type DummyJob struct {
	SimpleJob
	successProb float64 // Probability of success (0.0-1.0)
}

// NewDummyJob creates a new dummy job
func NewDummyJob(id, name string, schedType ScheduleType,
	workTime time.Duration, successProb float64) *DummyJob {

	job := &DummyJob{
		SimpleJob: SimpleJob{
			id:        id,
			name:      name,
			schedType: schedType,
			workTime:  workTime,
		},
		successProb: successProb,
	}

	// Set the work function
	job.SimpleJob.workFunc = job.dummyWork

	return job
}

// dummyWork is the work function for DummyJob
func (j *DummyJob) dummyWork(ctx context.Context) (string, error) {
	// Simulate some work by sleeping
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(j.workTime):
		// Determine success based on probability
		if rand.Float64() < j.successProb {
			return fmt.Sprintf("Job %s completed successfully", j.id), nil
		}
		return "", fmt.Errorf("job %s failed with simulated error", j.id)
	}
}

// LoggingJob is a job that logs messages at intervals
type LoggingJob struct {
	SimpleJob
	logIntervals []time.Duration // When to log messages
}

// NewLoggingJob creates a new logging job
func NewLoggingJob(id, name string, schedType ScheduleType,
	workTime time.Duration, logIntervals []time.Duration) *LoggingJob {

	job := &LoggingJob{
		SimpleJob: SimpleJob{
			id:        id,
			name:      name,
			schedType: schedType,
			workTime:  workTime,
		},
		logIntervals: logIntervals,
	}

	// Set the work function
	job.SimpleJob.workFunc = job.loggingWork

	return job
}

// loggingWork is the work function for LoggingJob
func (j *LoggingJob) loggingWork(ctx context.Context) (string, error) {
	// Create a ticker for each log interval
	type tickerInfo struct {
		ticker *time.Ticker
		msg    string
	}

	tickers := make([]*tickerInfo, len(j.logIntervals))
	for i, interval := range j.logIntervals {
		tickers[i] = &tickerInfo{
			ticker: time.NewTicker(interval),
			msg:    fmt.Sprintf("Log message at interval %v", interval),
		}
	}

	// Ensure tickers are stopped
	defer func() {
		for _, t := range tickers {
			t.ticker.Stop()
		}
	}()

	// Create a timer for the job duration
	timer := time.NewTimer(j.workTime)
	defer timer.Stop()

	// Process log messages
	logCount := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Sprintf("Job interrupted after %d log messages", logCount), ctx.Err()
		case <-timer.C:
			return fmt.Sprintf("Job completed after %d log messages", logCount), nil
		default:
			// Check all tickers
			for _, t := range tickers {
				select {
				case <-t.ticker.C:
					fmt.Println(t.msg)
					logCount++
				default:
					// Continue checking other tickers
				}
			}
			// Small sleep to prevent CPU spinning
			time.Sleep(10 * time.Millisecond)
		}
	}
}
