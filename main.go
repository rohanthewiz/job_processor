package main

import (
	"job_processor/jobpro"
	"job_processor/shutdown"
	"log"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

func main() {
	manager, store := jobpro.InitJobPro()
	defer func() {
		_ = store.Close()
	}()

	// done channel will signal when shutdown complete
	done := make(chan struct{})

	shutdown.InitShutdownSvc(done)

	// Close the job manager on shutdown
	shutdown.RegisterShutdownHook(func(gracePeriod time.Duration) error {
		err := manager.Shutdown(gracePeriod)
		if err != nil {
			logger.LogErr(err, "Error during job manager shutdown")
		} else {
			logger.Info("Job manager shutdown")
		}
		return err
	})

	// Register some example jobs
	if err := registerExampleJobs(manager); err != nil {
		logger.LogErr(err, "Failed to register example jobs")
	}

	// Block until done signal
	<-done
	log.Println("App exited")
}

// registerExampleJobs adds some example jobs to the manager
func registerExampleJobs(manager jobpro.JobMgr) error {
	// Create a new periodic job
	periodicJob := jobpro.NewPeriodicJob(
		"periodic-2",
		"Periodic Logging Job",
		0, 0, func() error {
			log.Println("Starting periodic action")
			return nil
		},
	)

	periodicID, err := manager.RegisterJob(periodicJob, "*/5 * * * * *")
	if err != nil {
		return serr.Wrap(err, "failed to create periodic job")
	}
	log.Printf("Created periodic job with ID: %s", periodicID)

	// Start the periodic job
	if err := manager.StartJob(periodicID); err != nil {
		logger.LogErr(serr.Wrap(err, "Failed to start periodic job"))
	}

	// // Create a periodic job that runs every minute
	// periodicJob := jobpro.NewLoggingJob(
	// 	"periodic-1",
	// 	"Periodic Logging Job",
	// 	jobpro.Periodic,
	// 	20*time.Second,
	// 	[]time.Duration{5 * time.Second},
	// )
	//
	// periodicID, err := manager.RegisterJob(periodicJob, "0 */1 * * * *") // Run every minute
	// if err != nil {
	// 	return serr.F("failed to create periodic job: %w", err)
	// }
	// log.Printf("Created periodic job with ID: %s", periodicID)
	//
	// // Start the periodic job
	// if err := manager.StartJob(periodicID); err != nil {
	// 	logger.LogErr(serr.Wrap(err, "Failed to start periodic job"))
	// }
	//

	return nil
}
