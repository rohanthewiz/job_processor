package jobpro

import (
	"context"
	"fmt"
	"job_processor/util"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// ScheduledJob is a job that logs messages at possibly multiple intervals
type ScheduledJob struct {
	BaseJob
	Call func() error // Function to call at each interval
}

// NewScheduledJob creates a new logging job
func NewScheduledJob(jc JobConfig) *ScheduledJob {
	job := &ScheduledJob{
		BaseJob: BaseJob{
			id:          jc.ID,
			name:        jc.Name,
			freqType:    util.If(jc.IsPeriodic, Periodic, OneTime),
			maxWorkTime: time.Duration(jc.MaxRunTime) * time.Second,
		},
		Call: jc.JobFunction,
	}

	// Set the work function
	job.BaseJob.workFunc = job.scheduledRun

	return job
}

// scheduledRun is the work function for ScheduledJob overriding the base job's Run
func (j *ScheduledJob) scheduledRun(ctx context.Context) (results string, err error) {
	jobTypeName := util.If(j.freqType == Periodic, "Periodic", "Onetime")

	// Create a timer for the job duration only if maxWorkTime > 0
	var timer *time.Timer
	if j.maxWorkTime > 0 {
		timer = time.NewTimer(j.maxWorkTime)
		defer timer.Stop()
	}

	select {
	case <-ctx.Done():
		return "Job interrupted before execution", ctx.Err()

	case <-j.timerOrNil(timer):
		return "Job timed out before execution", nil

	default:
		// Execute once
		fmt.Printf("Running %s job: %s\n", jobTypeName, j.name)
		err = j.Call()
		if err != nil {
			ser := serr.Wrap(err)
			logger.LogErr(ser, "Error executing %s job", j.name)
			return results, ser
		}

		return fmt.Sprintf("%s job %s, completed", jobTypeName, j.name), nil
	}
}

// RunAt will run a function at a certain time and return a timer
// whose Stop function we can call (we should not use the Ticker of the returned timer though)
func RunAt(at time.Time, fn func()) (AfterFuncTimer *time.Timer) {
	// Run after a duration of now until `at`
	return time.AfterFunc(time.Until(at), fn)
}
