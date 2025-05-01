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

	// Modern styled version of the Jobs Management HTML generator
	b.Html().R(
		b.Head().R(
			b.Title().R("Jobs Management"),
			// Add meta tag for responsive design
			b.T(`<meta name="viewport" content="width=device-width, initial-scale=1.0">`),
			// Add modern CSS styling
			b.Style().T(`
:root {
	--primary-color: #4a6cf7;
	--secondary-color: #607D8B;
	--background-color: #F2F7F7;
	--header-bg: #DCE6E6;
	--border-color: #C8D5D5;
	--success-color: #42A99A;
	--warning-color: #BF9A56;
	--danger-color: #D06862;
}

body {
	font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
	line-height: 1.5;
	color: #3D5656;
	background-color: var(--background-color);
	margin: 0;
	padding: 20px;
}

h1 {
	color: #2E454B;
	margin-bottom: 1.5rem;
	font-weight: 600;
}

.container {
	max-width: 1200px;
	margin: 0 auto;
	padding: 1rem;
	background-color: #FBFEFE;
	border-radius: 8px;
	box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.table-responsive {
	overflow-x: auto;
	margin-bottom: 1rem;
}

table {
	width: 100%;
	border-collapse: collapse;
	margin-bottom: 1rem;
	font-size: 0.875rem;
}

th {
	background-color: var(--header-bg);
	padding: 0.75rem 1rem;
	text-align: left;
	font-weight: 600;
	color: #2E454B;
	border-bottom: 2px solid var(--border-color);
}

td {
	padding: 0.75rem 1rem;
	border-bottom: 1px solid var(--border-color);
	vertical-align: top;
}

tr:last-child td {
	border-bottom: none;
}

tr:hover {
	background-color: rgba(220, 230, 230, 0.6);
}

.badge {
	display: inline-block;
	padding: 0.25rem 0.5rem;
	border-radius: 9999px;
	font-size: 0.75rem;
	font-weight: 500;
	text-transform: uppercase;
	letter-spacing: 0.05em;
}

.badge-active {
	background-color: rgba(66, 169, 154, 0.15);
	color: var(--success-color);
}

.badge-pending {
	background-color: rgba(191, 154, 86, 0.15);
	color: var(--warning-color);
}

.badge-inactive {
	background-color: rgba(96, 125, 139, 0.15);
	color: var(--secondary-color);
}

.badge-error {
	background-color: rgba(208, 104, 98, 0.15);
	color: var(--danger-color);
}

.timestamp {
	font-family: monospace;
	font-size: 0.75rem;
	color: var(--secondary-color);
}

@media (max-width: 768px) {
	th, td {
		padding: 0.5rem;
	}
}
`),
		),
		b.Body().R(
			b.Div("class", "container").R(
				b.H1().T("Jobs Management"),
				b.Div("class", "table-responsive").R(
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
									b.Wrap(func() {
										if !job.NextRunTime.IsZero() && job.NextRunTime.After(time.Now()) {
											b.Td("class", "timestamp").T(job.NextRunTime.Format("2006-01-02 15:04 MST"))
										} else {
											b.Td().T("N/A")
										}
									}),
									b.Wrap(func() {
										statusClass := "badge badge-inactive"
										switch job.Status {
										case "active":
											statusClass = "badge badge-active"
										case "pending":
											statusClass = "badge badge-pending"
										case "error":
											statusClass = "badge badge-error"
										}
										b.Td().R(
											b.Span("class", statusClass).T(string(job.Status)),
										)
									}),
									b.Td("class", "timestamp").T(job.CreatedAt.Format("2006-01-02 15:04 MST")),
									b.Td("class", "timestamp").T(job.UpdatedAt.Format("2006-01-02 15:04 MST")),
								)
							}),
						),
					),
				),
			),
		),
	)

	return b.String()
}
