package main

import (
	"fmt"
	"job_processor/jobpro"
	"job_processor/shutdown"
	"log"
	"os"
	"time"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
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

	return nil
}

func rootHandler(ctx rweb.Context) error {
	return ctx.WriteJSON(map[string]interface{}{
		"response": "OK",
		"ENV":      os.Getenv("ENV"),
	})
}

func renderJobsTable(jobs []jobpro.JobDef) string {
	b := element.NewBuilder()
	cols := []string{"JobID", "JobName", "JobType", "NextRun", "Status", "CreatedAt", "UpdatedAt"}

	b.Html().R(
		b.Head().R(
			b.Title().R("Jobs Mgmt"),
		),
		b.Body().R(
			b.Table().R(
				b.Head().R(
					b.Tr().R(
						element.ForEach(cols, func(col string) {
							b.Th().T(col)
						}),
					),
				),
				b.TBody().R(
					element.ForEach(jobs, func(job jobpro.JobDef) {
						b.Tr().R(
							b.Td().T(job.JobID),
							b.Td().T(job.JobName),
							b.Td().T(string(job.SchedType)),
							b.Td().T(job.NextRunTime.Format(time.RFC3339)),
							b.Td().T(string(job.Status)),
							b.Td().T(job.CreatedAt.Format(time.RFC3339)),
							b.Td().T(job.UpdatedAt.Format(time.RFC3339)),
						)
					}),
				),
			),
		),
	)

	return b.String()
}
