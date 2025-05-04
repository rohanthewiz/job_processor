package main

import (
	"fmt"
	"job_processor/jobpro"
	"job_processor/shutdown"
	"log"
	"os"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

func main() {
	manager, store := jobpro.Init("jobs.ddb")
	defer func() {
		_ = store.Close()
	}()

	// done channel will signal when shutdown complete
	done := make(chan struct{})
	shutdown.InitService(done)

	// Close the job manager on shutdown
	shutdown.RegisterHook(func(gracePeriod time.Duration) error {
		err := manager.Shutdown(gracePeriod)
		if err != nil {
			logger.LogErr(err, "Error during job manager shutdown")
		} else {
			logger.Info("Job manager shutdown")
		}
		return err
	})

	if err := registerJobs(manager); err != nil {
		logger.LogErr(err, "Failed to register jobs")
	}

	// Inline WebServe so we can see the job manager etc
	go func() {
		s := rweb.NewServer(rweb.ServerOptions{
			Address: fmt.Sprintf(":%s", "8800"),
			Verbose: true,
		})

		s.Use(rweb.RequestInfo)

		s.Get("/", rootHandler)

		s.Get("/show-jobs", func(ctx rweb.Context) error {
			jobs, err := manager.ListJobs()
			if err != nil {
				logger.LogErr(err, "Failed to list jobs")
				return serr.Wrap(err)
			}
			return ctx.WriteHTML(renderJobsTable(jobs))
		})

		log.Println(s.Run())
	}()

	// Block until done signal
	<-done
	log.Println("App exited")
}

// registerJobs adds some example jobs to the manager
func registerJobs(manager jobpro.JobMgr) error {
	// Create a new periodic job
	periodicJob := jobpro.NewPeriodicJob(
		"periodic-2",
		"Periodic Job",
		0, 0, func() error {
			log.Println("Starting periodic action")
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

func rootHandler(ctx rweb.Context) error {
	return ctx.WriteJSON(map[string]interface{}{
		"response": "OK",
		"ENV":      os.Getenv("ENV"),
	})
}
