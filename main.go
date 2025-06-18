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
	"github.com/rohanthewiz/serr"
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

	registerJobs(jobMgr)

	// Block until done signal
	<-done
	fmt.Println("App exited")
}

// registerJobs configures and registers job definitions with the job manager.
// It attempts to load job configurations from an HTTP endpoint, falling back to
// hardcoded configurations if the fetch fails. Each job is mapped to its
// corresponding function and registered with the job manager.
func registerJobs(jobMgr *jobpro.DefaultJobManager) {
	// Testing LifeCycle of Streamsets job
	// jobpro.RegisterJob(jobpro.JobConfig{
	// 	ID:          "SS-API-Job",
	// 	Name:        "Streamsets API Job",
	// 	IsPeriodic:  true,
	// 	Schedule:    "*/30 * * * * *",
	// 	AutoStart:   true,
	// 	JobFunction: streamsets.RunJob,
	// })

	// Define what functions each job should run.
	// This is for demo purposes -- for app we could call a local or endpoint to do some work
	jobFunctions := map[string]func() error{
		"periodicJob1": func() error {
			fmt.Println("Periodic job1 doing work")
			return nil
		},
		"periodicJob2": func() error {
			fmt.Println("Periodic job2 attempting work...")
			return serr.New("Testing error return")
		},
		"onetimeJob1": func() error {
			fmt.Println("One time job doing work")
			return nil
		},
		"manualJob": func() error {
			fmt.Println("Manual start job doing work")
			return nil
		},
	}

	// Fetch job configs from endpoint
	configs, err := jobpro.FetchJobConfigs("http://localhost:8000/job/config/job_configs.json")
	if err != nil {
		logger.LogErr(err, "Failed to fetch job configs, exiting..")
		os.Exit(1)
		// Hardcoded configs
		// jobpro.RegisterJob(jobpro.JobConfig{
		// 	ID:          "periodicJob1",
		// 	Name:        "Periodic Job 1",
		// 	IsPeriodic:  true,
		// 	Schedule:    "*/15 * * * * *",
		// 	AutoStart:   true,
		// 	JobFunction: jobFunctions["periodicJob1"],
		// })
		//
		// jobpro.RegisterJob(jobpro.JobConfig{
		// 	ID:          "onetimeJob1",
		// 	Name:        "Onetime Job 1",
		// 	IsPeriodic:  false,
		// 	Schedule:    "in 30s", // Can also use: "+30s", "30s", "2024-01-15 14:30:00 PST", etc. See README
		// 	AutoStart:   true,     // Auto-start this job
		// 	JobFunction: jobFunctions["onetimeJob1"],
		// })
		//
		// jobpro.RegisterJob(jobpro.JobConfig{
		// 	ID:          "manualJob",
		// 	Name:        "Manual Start Job",
		// 	IsPeriodic:  false,
		// 	Schedule:    "",
		// 	AutoStart:   false, // This job won't start automatically
		// 	JobFunction: jobFunctions["manualJob"],
		// })
	}

	// Register jobs from configs
	logger.F("%d job configurations received from JSON file", len(configs))
	for _, config := range configs {
		jobFunc, exists := jobFunctions[config.ID]
		if !exists {
			logger.Log("Warning: No job function found for job ID", "id", config.ID)
			continue
		}

		// Add the job function to the config
		config.JobFunction = jobFunc

		// Register the job
		jobpro.RegisterJob(config)
	}

	if err := jobpro.LoadJobs(jobMgr); err != nil {
		logger.LogErr(err, "Failed to load jobs into job manager. Exiting...")
		os.Exit(1)
	}
}
