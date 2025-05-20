package main

import (
	"job_processor/jobpro"
	"log"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// registerJobs adds some example jobs to the manager
func registerJobs(manager jobpro.JobMgr) error {
	// Create a new periodic job
	periodicJob := jobpro.NewPeriodicJob(
		"per-2",
		"Periodic Job 2",
		0, 0,
		func() error {
			// Do Work
			log.Println("Starting periodic action per-2")
			return nil
		},
	)

	periodicID, err := manager.RegisterJob(periodicJob, "*/10 * * * * *")
	if err != nil {
		return serr.Wrap(err, "failed to create periodic job")
	}
	log.Printf("Created periodic job with ID: %s", periodicID)

	// Start the periodic job
	if err := manager.StartJob(periodicID); err != nil {
		logger.LogErr(serr.Wrap(err, "Failed to start periodic job"))
	}

	return nil
}
