package jobpro

import (
	"context"
	"fmt"
	"time"
)

// LoggingJob is a job that logs messages at possibly multiple intervals
// It is not recommened to use this  as it is adding timing on top of a cron system
// better to stick to the standard ScheduledJob
type LoggingJob struct {
	BaseJob
	logIntervals []time.Duration // When to log messages
}

// NewLoggingJob creates a new logging job
func NewLoggingJob(id, name string, schedType FreqType,
	workTime time.Duration, logIntervals []time.Duration) *LoggingJob {

	job := &LoggingJob{
		BaseJob: BaseJob{
			id:          id,
			name:        name,
			freqType:    schedType,
			maxWorkTime: workTime,
		},
		logIntervals: logIntervals,
	}

	// Set the work function
	job.BaseJob.workFunc = job.loggingWork

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
	timer := time.NewTimer(j.maxWorkTime)
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
