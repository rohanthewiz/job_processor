package jobpro

import (
	"context"
	"fmt"
	"time"
)

// PeriodicJob is a job that logs messages at possibly multiple intervals
type PeriodicJob struct {
	BaseJob
	// Period time.Duration // Allows an additional interval on top of cron -- not using for now
	Call func() error // Function to call at each interval -- overrides BaseJob.WorkFunc
}

// NewPeriodicJob creates a new logging job
func NewPeriodicJob(jc JobConfig) *PeriodicJob {

	job := &PeriodicJob{
		BaseJob: BaseJob{
			id:          jc.ID,
			name:        jc.Name,
			freqType:    Periodic,
			maxWorkTime: 0,
		},
		Call: jc.JobFunction,
	}

	// Set the work function
	job.BaseJob.workFunc = job.periodicRun

	return job
}

// periodicRun is the work function for PeriodicJob overriding the base job's Run
func (j *PeriodicJob) periodicRun(ctx context.Context) (results string, err error) {
	// Create a timer for the job duration only if maxWorkTime > 0
	var timer *time.Timer
	if j.maxWorkTime > 0 {
		timer = time.NewTimer(j.maxWorkTime)
		defer timer.Stop()
	}

	runCount := 0

	/*	if j.Period > 0 { // If Period is set, use a ticker -- let's not do this for now
		ticker := time.NewTicker(j.Period)
		defer ticker.Stop()

		msg := fmt.Sprintf("Process Catalog trigger %v", j.Period)

		for {
			select {
			case <-ctx.Done():
				return fmt.Sprintf("Job interrupted after %d runs", runCount), ctx.Err()

			// Only include timer case when maxWorkTime > 0
			case <-j.timerOrNil(timer):
				return fmt.Sprintf("Job shutdown after %d runs", runCount), nil

			case <-ticker.C:
				fmt.Println(msg)
				err = j.Call()
				runCount++
				// Do a trigger here
			}
			// Small sleep to prevent CPU spinning
			time.Sleep(100 * time.Millisecond)
		}
	} else {*/
	// No Period set, just run once
	select {
	case <-ctx.Done():
		return "Job interrupted before execution", ctx.Err()

	case <-j.timerOrNil(timer):
		return "Job timed out before execution", nil

	default:
		// Execute once
		fmt.Println("Running periodic job")
		err = j.Call()
		runCount++

		return fmt.Sprintf("Periodic job completed with %d run(s)", runCount), nil
	}
	// }
}
