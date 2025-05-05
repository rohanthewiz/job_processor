package main

import (
	"fmt"
	"job_processor/jobpro"
	"job_processor/pubsub"
	"job_processor/shutdown"
	"log"
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

	if err := pubsub.StartPubSub(); err != nil {
		logger.LogErr(err, "Failed to start pubsub")
		os.Exit(1)
	}

	if err := pubsub.ListenForUpdates(jobMgr.GetJobsUpdatedChan()); err != nil {
		logger.LogErr(err, "Failed to setup listener for job updates")
	}

	if err := registerJobs(jobMgr); err != nil {
		logger.LogErr(err, "Failed to register jobs")
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

		s.Get("/jobs-table-rows", func(ctx rweb.Context) error {
			jobs, err := jobMgr.ListJobs()
			if err != nil {
				logger.LogErr(err, "Failed to list jobs")
				return serr.Wrap(err)
			}

			b := element.NewBuilder()
			renderJobsTableRows(b, jobs)

			return ctx.WriteHTML(b.String())
		})

		s.Get("/jobs-update", func(ctx rweb.Context) error {
			fmt.Println("In SSE handler")
			out := make(chan any, 1)
			_, err := pubsub.SubscribeToUpdates(out)
			if err != nil {
				return serr.Wrap(err)
			}

			s.SetupSSE(ctx, out, "job-update")

			// fmt.Println("Exiting SSE handler")
			// sub.Unsubscribe()
			return nil
		})

		log.Println(s.Run())
	}()

	// Block until done signal
	<-done
	println("App exited")
}

func rootHandler(ctx rweb.Context) error {
	return ctx.WriteJSON(map[string]interface{}{
		"response": "OK",
		"ENV":      os.Getenv("ENV"),
	})
}
