package main

import (
	"fmt"
	"job_processor/jobpro"
	"job_processor/pubsub"
	"job_processor/shutdown"
	"job_processor/web"
	"os"

	"github.com/rohanthewiz/logger"
)

func main() {
	done := make(chan struct{}) // done channel will signal when shutdown complete
	shutdown.InitShutdownService(done)

	jobMgr := jobpro.Init("jobs.ddb")

	// Start PubSub so UI can receive SSE events
	if err := pubsub.StartPubSub(); err != nil {
		logger.LogErr(err, "Failed to start pubsub")
		os.Exit(1)
	}

	if err := pubsub.ListenForUpdates(jobMgr.GetJobsUpdatedChan()); err != nil {
		logger.LogErr(err, "Failed to setup listener for job updates")
	}

	registerJobs(jobMgr)

	go web.StartWebServer(jobMgr)

	// Block until done signal
	<-done
	fmt.Println("App exited")
}

func registerJobs(jobMgr *jobpro.DefaultJobManager) {
	// Register a job - we should be able to register jobs from any package.
	jobpro.RegisterJob(jobpro.JobConfig{
		ID:         "periodicJob1",
		Name:       "Periodic Job 1",
		IsPeriodic: true,
		Schedule:   "*/15 * * * * *",
		AutoStart:  true,
		JobFunction: func() error {
			fmt.Println("Periodic job doing work")
			return nil
		},
	})

	// Register a Onetime job - now supports multiple time formats!
	jobpro.RegisterJob(jobpro.JobConfig{
		ID:         "onetimeJob1",
		Name:       "Onetime Job 1",
		IsPeriodic: false,
		Schedule:   "in 30s", // Can also use: "+30s", "30s", "2024-01-15 14:30:00 PST", etc. See README
		AutoStart:  true,     // Auto-start this job
		JobFunction: func() error {
			fmt.Println("One time job doing work")
			return nil
		},
	})

	// Register another Onetime job that doesn't auto-start for testing
	jobpro.RegisterJob(jobpro.JobConfig{
		ID:         "onetimeJob2",
		Name:       "Manual Start Job",
		IsPeriodic: false,
		Schedule:   "",
		AutoStart:  false, // This job won't start automatically
		JobFunction: func() error {
			fmt.Println("Manual start job doing work")
			return nil
		},
	})

	if err := jobpro.LoadJobs(jobMgr); err != nil {
		logger.LogErr(err, "Failed to load jobs")
		os.Exit(1)
	}
}
