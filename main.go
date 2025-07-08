// Package main provides the entry point for the job processor application,
// coordinating job management, web server, and pubsub components.
package main

import (
	"fmt"
	"job_processor/jobpro"
	"job_processor/pubsub"
	"job_processor/shutdown"
	"job_processor/web"
	"os"
	"time"

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

	// Start web server before registering jobs so the endpoint is available
	go web.StartWebServer(jobMgr)

	// Give the web server a moment to start
	time.Sleep(500 * time.Millisecond)

	// Fetch jobs

	registerJobs(jobMgr)

	// Block until done signal
	<-done
	fmt.Println("App exited")
}

// registerJobs configures and registers job definitions with the job manager.
// It attempts to load job configurations from an HTTP endpoint "/jobs/definitions" of the secondary container
func registerJobs(jobMgr *jobpro.DefaultJobManager) {
	endpoint := jobpro.BackendURLWoPath() + "/jobs/definitions"

	jobConfigs, err := jobpro.FetchJobConfigs(endpoint)
	if err != nil {
		logger.LogErr(err, "Failed to fetch job configs, exiting..")
		os.Exit(1)
	}

	logger.F("%d job configurations received from backend container", len(jobConfigs))

	// REGISTER JOBS FROM CONFIGS
	for _, config := range jobConfigs {
		// Register the job
		jobpro.RegisterJob(config)
	}

	if err := jobpro.LoadJobs(jobMgr); err != nil {
		logger.LogErr(err, "Failed to load jobs into job manager. Exiting...")
		os.Exit(1)
	}

	// Hardcoded jobConfigs
	// jobpro.RegisterJob(jobpro.JobConfig{
	// 	Id:          "periodicJob1",
	// 	Name:        "Periodic Job 1",
	// 	IsPeriodic:  true,
	// 	Schedule:    "*/15 * * * * *",
	//  TriggerEndpoint string // activate the job on the backend container
	// 	AutoStart:   true,
	// 	JobFunction: jobFunctions["periodicJob1"],
	// })
	//
	// jobpro.RegisterJob(jobpro.JobConfig{
	// 	Id:          "onetimeJob1",
	// 	Name:        "Onetime Job 1",
	// 	IsPeriodic:  false,
	// 	Schedule:    "in 30s", // Can also use: "+30s", "30s", "2024-01-15 14:30:00 PST", etc. See README
	//  TriggerEndpoint string // activate the job on the backend container
	// 	AutoStart:   true,     // Auto-start this job
	// 	JobFunction: jobFunctions["onetimeJob1"],
	// })
	//
	// jobpro.RegisterJob(jobpro.JobConfig{
	// 	Id:          "manualJob",
	// 	Name:        "Manual Start Job",
	// 	IsPeriodic:  false,
	// 	Schedule:    "",
	//  TriggerEndpoint string // activate the job on the backend container
	// 	AutoStart:   false, // This job won't start automatically
	// 	JobFunction: jobFunctions["manualJob"],
	// })
}
