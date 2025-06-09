package main

import (
	"encoding/json"
	"fmt"
	"job_processor/jobpro"
	"job_processor/pubsub"
	"job_processor/shutdown"
	"os"
	"strconv"

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
		ID:         "periodicJob1",
		Name:       "Periodic Job 1",
		IsPeriodic: true,
		Schedule:   "*/10 * * * * *",
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
		Schedule:   "in 2m",
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

	go func() { // Todo move this to "web" package
		s := rweb.NewServer(rweb.ServerOptions{
			Address: fmt.Sprintf(":%s", "8000"),
			Verbose: true,
		})

		s.Use(rweb.RequestInfo)

		s.Get("/", rootHandler)

		s.Get("/jobs", func(ctx rweb.Context) error {
			jobs, resultCounts, err := jobMgr.ListJobsWithPagination(10)
			if err != nil {
				logger.LogErr(err, "Failed to list jobs")
				return serr.Wrap(err)
			}
			return ctx.WriteHTML(renderJobsTable(jobs, resultCounts))
		})

		// Endpoint to get the jobs table rows
		// Typically this is called after an SSE event is received on job update
		s.Get("/jobs/get-table-rows", func(ctx rweb.Context) error {
			jobs, resultCounts, err := jobMgr.ListJobsWithPagination(10)
			if err != nil {
				logger.LogErr(err, "Failed to list jobs")
				return serr.Wrap(err) // guaranteed
			}

			b := element.NewBuilder()
			renderJobsTableRows(b, jobs, resultCounts)

			return ctx.WriteHTML(b.String())
		})

		// Get more results for a specific job
		s.Get("/jobs/results/:job-id", func(ctx rweb.Context) error {
			jobID := ctx.Request().Param("job-id")

			// Get offset from query parameter
			offsetStr := ctx.Request().QueryParam("offset")
			offset := 0
			if offsetStr != "" {
				if val, err := strconv.Atoi(offsetStr); err == nil {
					offset = val
				}
			}

			results, totalCount, err := jobMgr.GetJobResultsPaginated(jobID, offset, 10)
			if err != nil {
				logger.LogErr(err, "Failed to get job results", "jobID", jobID)
				ctx.Status(500)
				return ctx.WriteJSON(map[string]string{
					"error": err.Error(),
				})
			}

			// Render result rows as HTML
			b := element.NewBuilder()
			for i, result := range results {
				// Calculate actual run number: most recent run has highest number
				runNumber := totalCount - offset - i
				b.Tr("class", fmt.Sprintf("job-result-row job-%s", jobID), "data-job-id", jobID, "style", "display: none;").R(
					b.Td().T(""),    // Empty for job name
					b.Td().T(jobID), // Job ID
					b.Td().T(""),    // Empty for frequency
					b.Td().T(""),    // Empty for status
					b.Td().T(""),    // Empty for created
					b.Td().T(""),    // Empty for updated
					b.Td().F("#%d", runNumber),
					b.TdClass("timestamp").T(result.StartTime.Format("2006-01-02 15:04 MST")),
					b.Td().F("%0.1f ms", float64(result.Duration.Microseconds())/1000),
					b.Td().T(string(result.Status)),
					b.Td().T(result.ErrorMsg),
					b.Td().T(""), // Empty controls column for result rows
				)
			}

			// Add a load more button if there are more results
			if offset+len(results) < totalCount {
				b.Tr("class", fmt.Sprintf("load-more-row job-%s", jobID), "style", "display: none;").R(
					b.Td("colspan", "12", "style", "text-align: center; padding: 10px;").R(
						b.Button("class", "btn btn-secondary load-more-btn",
							"data-job-id", jobID,
							"data-offset", fmt.Sprintf("%d", offset+10),
							"data-total", fmt.Sprintf("%d", totalCount),
							"onclick", "loadMoreResults(this)").F(
							"Load %d more (showing %d of %d)",
							min(10, totalCount-(offset+len(results))),
							offset+len(results),
							totalCount,
						),
					),
				)
			}

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

		s.Post("/jobs/start/:job-id", func(ctx rweb.Context) error {
			jobID := ctx.Request().Param("job-id")

			if err := jobMgr.StartJob(jobID); err != nil {
				logger.LogErr(err, "Failed to start job", "jobID", jobID)
				ctx.Status(500)
				return ctx.WriteJSON(map[string]string{
					"error": err.Error(),
				})
			}

			return ctx.WriteJSON(map[string]string{
				"jobID":  jobID,
				"status": "started",
			})
		})

		s.Post("/jobs/stop/:job-id", func(ctx rweb.Context) error {
			jobID := ctx.Request().Param("job-id")

			if err := jobMgr.StopJob(jobID); err != nil {
				logger.LogErr(err, "Failed to stop job", "jobID", jobID)
				ctx.Status(500)
				return ctx.WriteJSON(map[string]string{
					"error": err.Error(),
				})
			}

			return ctx.WriteJSON(map[string]string{
				"jobID":  jobID,
				"status": "stopped",
			})
		})

		s.Post("/jobs/reschedule/:job-id", func(ctx rweb.Context) error {
			jobID := ctx.Request().Param("job-id")

			// Parse request body to get new schedule
			type rescheduleRequest struct {
				Schedule string `json:"schedule"`
			}
			var req rescheduleRequest

			// Read body and decode JSON
			bodyBytes := ctx.Request().Body()
			if err := json.Unmarshal(bodyBytes, &req); err != nil {
				ctx.Status(400)
				return ctx.WriteJSON(map[string]string{
					"error": "Invalid request: " + err.Error(),
				})
			}

			if req.Schedule == "" {
				ctx.Status(400)
				return ctx.WriteJSON(map[string]string{
					"error": "Schedule is required",
				})
			}

			if err := jobMgr.RescheduleJob(jobID, req.Schedule); err != nil {
				logger.LogErr(err, "Failed to reschedule job", "jobID", jobID)
				ctx.Status(500)
				return ctx.WriteJSON(map[string]string{
					"error": err.Error(),
				})
			}

			return ctx.WriteJSON(map[string]string{
				"jobID":    jobID,
				"status":   "rescheduled",
				"schedule": req.Schedule,
			})
		})

		// Get job history for charts
		s.Get("/jobs/history/:job-id", func(ctx rweb.Context) error {
			jobID := ctx.Request().Param("job-id")

			results, err := jobMgr.GetJobHistory(jobID, 10) // Get last 10 runs
			if err != nil {
				logger.LogErr(err, "Failed to get job history", "jobID", jobID)
				ctx.Status(500)
				return ctx.WriteJSON(map[string]string{
					"error": err.Error(),
				})
			}

			return ctx.WriteJSON(results)
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
