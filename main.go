package main

import (
	"fmt"
	"job_processor/jobpro"
	"job_processor/pubsub"
	"job_processor/shutdown"
	"os"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
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

	// Register a job - we should be able to register jobs from any package.
	jobpro.RegisterJob(jobpro.JobConfig{
		ID:         "job1",
		Name:       "Example Job 1",
		IsPeriodic: true,
		Schedule:   "*/15 * * * * *", // Every 10s
		JobFunction: func() error {
			fmt.Println("doing work")
			return nil
		},
	})

	if err := jobpro.LoadJobs(jobMgr); err != nil {
		logger.LogErr(err, "Failed to load jobs")
		os.Exit(1)
	}

	go func() {
		s := rweb.NewServer(rweb.ServerOptions{
			Address: fmt.Sprintf(":%s", "8000"),
			Verbose: true,
		})

		s.Use(rweb.RequestInfo)

		s.Get("/", rootHandler)

		s.Get("/jobs", func(ctx rweb.Context) error {
			jobs, err := jobMgr.ListJobs()
			if err != nil {
				logger.LogErr(err, "Failed to list jobs")
				return serr.Wrap(err)
			}
			return ctx.WriteHTML(renderJobsTable(jobs))
		})

		// Endpoint to get the jobs table rows
		// Typically this is called after an SSE event is received on job update
		s.Get("/jobs/get-table-rows", func(ctx rweb.Context) error {
			jobs, err := jobMgr.ListJobs()
			if err != nil {
				logger.LogErr(err, "Failed to list jobs")
				return serr.Wrap(err) // guaranteed
			}

			b := element.NewBuilder()
			renderJobsTableRows(b, jobs)

			return ctx.WriteHTML(b.String())
		})

		// SSE endpoint for job updates
		s.Get("/jobs/update-notify", func(ctx rweb.Context) error {
			fmt.Println("Handling SSE request")
			out := make(chan any, 1)
			_, err := pubsub.SubscribeToUpdates(out)
			if err != nil {
				return serr.Wrap(err)
			}

			// Remember that this is just the setup of the SSE connection headers etc.
			// Data will flow *after* this function exits
			err = s.SetupSSE(ctx, out, "job-update")
			if err != nil {
				err = serr.Wrap(err)
			}
			return err
		})

		s.Post("/jobs/pause/:job-id", func(ctx rweb.Context) error {
			jobID := ctx.Request().Param("job-id")

			// Assume your jobpro.Manager has a PauseJob method
			if err := jobMgr.PauseJob(jobID); err != nil {
				logger.LogErr(err, "Failed to pause job", "jobID", jobID)
				ctx.Status(500)
				return ctx.WriteJSON(map[string]string{
					"error": err.Error(),
				})
			}

			return ctx.WriteJSON(map[string]string{
				"jobID":  jobID,
				"status": "paused",
			})
		})

		s.Post("/jobs/resume/:job-id", func(ctx rweb.Context) error {
			jobID := ctx.Request().Param("job-id")

			if err := jobMgr.ResumeJob(jobID); err != nil {
				logger.LogErr(err, "Failed to resume job", "jobID", jobID)
				ctx.Status(500)
				return ctx.WriteJSON(map[string]string{
					"error": err.Error(),
				})
			}

			return ctx.WriteJSON(map[string]string{
				"jobID":  jobID,
				"status": "resumed",
			})
		})

		s.Post("/jobs/run-now/:job-id", func(ctx rweb.Context) error {
			jobID := ctx.Request().Param("job-id")

			if err := jobMgr.TriggerJobNow(jobID); err != nil {
				logger.LogErr(err, "Failed to trigger job", "jobID", jobID)
				ctx.Status(500)
				return ctx.WriteJSON(map[string]string{
					"error": err.Error(),
				})
			}

			return ctx.WriteJSON(map[string]string{
				"jobID":  jobID,
				"status": "triggered",
			})
		})

		// Run the server
		err := s.Run()
		if err != nil {
			logger.LogErr(err, "where", "at server exit")
		}
	}()

	// Block until done signal
	<-done
	fmt.Println("App exited")
}

func rootHandler(ctx rweb.Context) error {
	return ctx.WriteJSON(map[string]interface{}{
		"response": "OK",
		"ENV":      os.Getenv("ENV"),
	})
}
