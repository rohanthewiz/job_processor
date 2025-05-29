package main

import (
	_ "embed"
	"job_processor/jobpro"
	"strings"

	"github.com/rohanthewiz/element"
)

//go:embed assets/table_styles.css
var tableStyles string

const jobEvent = "job-update"

// renderJobsTable renders the full jobs table page
func renderJobsTable(jobs []jobpro.JobRun) string {
	b := element.NewBuilder()
	cols := []string{"Job", "ID", "Freq", "Status", "Created", "Updated",
		"Run&nbsp;ID", "Run Start", "Duration", "Status", "Error", "Controls"}

	b.Html().R(
		b.Head().R(
			b.Title().T("Jobs"),
			// Add meta tag for responsive design
			b.T(`<meta name="viewport" content="width=device-width, initial-scale=1.0">`),
			b.Style().T(tableStyles),
			// Add HTMX library
			b.T(`<script src="https://unpkg.com/htmx.org@2.0.4"></script>`),
			// Add HTMX SSE extension
			b.T(`<script src="https://unpkg.com/htmx-ext-sse@2.2.2"></script>`),
		),
		b.Body().R(
			// Add SSE source connection to the body
			b.DivClass("container").R(
				b.H1().T("Jobs"),
				b.DivClass("table-responsive").R(
					b.Table().R(
						b.THead().R(
							b.Tr().R(
								element.ForEach(cols, func(col string) {
									b.Th().T(col)
								}),
							),
						),
						b.TBody("id", "jobs-table-body",
							"hx-ext", "sse", "sse-connect", "/jobs/update-notify",
							"hx-trigger", "sse:"+jobEvent,
							"hx-get", "/jobs/get-table-rows",
							"hx-swap", "innerHTML").R( // It seems best to do the SSE Swap on the immediate children

							renderJobsTableRows(b, jobs),
						),
					),
				),
			),
		),
	)

	return b.String()
}

// renderJobsTableRows renders just the table rows - for HTMX updates
func renderJobsTableRows(b *element.Builder, jobs []jobpro.JobRun) (x any) {
	element.ForEach(jobs, func(job jobpro.JobRun) {
		b.Tr().R(
			b.Td().T(job.JobName),
			b.Td().T(job.JobID),
			b.Wrap(func() {
				// Some Job level attributes
				if job.ResultId == 0 { // main job
					b.TdClass("cron").T(job.FreqType)

					statusClass := "badge badge-inactive"
					switch strings.ToLower(job.JobStatus) {
					case "running":
						statusClass = "badge badge-active"
					case "paused":
						statusClass = "badge badge-pending"
					case "pending":
						statusClass = "badge badge-pending"
					case "error":
						statusClass = "badge badge-error"
					}

					b.Td().R(
						b.SpanClass(statusClass).T(job.JobStatus),
					)
					b.TdClass("timestamp").T(job.CreatedAt.Format("2006-01-02 15:04 MST"))
					b.TdClass("timestamp").T(job.UpdatedAt.Format("2006-01-02 15:04 MST"))

				} else { // no need to display job level things for each run
					b.Td().T("")
					b.Td().T("")
					b.Td().T("")
					b.Td().T("")
				}

				// Some Run level attributes
				if job.ResultId == 0 { // no need to display runlevel things for the main job
					b.Td().T("")
					b.Td().T("")
					b.Td().T("")
					b.Td().T("")
					b.Td().T("")
					// Controls
					b.Td().R(
						b.DivClass("btn-group").R(
							// Play button
							b.AClass("btn btn-primary", "data-job-id", job.JobID, "title", "Resume Job", "onClick",
								`fetch('/jobs/resume/' + this.getAttribute('data-job-id'), {method: 'POST'})
                .then(response => {
                    if (response.ok) return response.json();
                    throw new Error('Network response was not ok');
                })
                .then(data => console.log('Job resumed:', data))
                .catch(error => console.error('Error resuming job:', error))`).R(
								b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
                xmlns="http://www.w3.org/2000/svg"
                style="vertical-align: middle;">
                <polygon points="5,4 15,10 5,16" fill="currentColor"/>
                </svg>`),
							),
							// Pause button
							b.AClass("btn btn-primary", "data-job-id", job.JobID, "title", "Pause Job", "onClick",
								`fetch('/jobs/pause/' + this.getAttribute('data-job-id'), {method: 'POST'})
                .then(response => {
                    if (response.ok) return response.json();
                    throw new Error('Network response was not ok');
                })
                .then(data => console.log('Job paused:', data))
                .catch(error => console.error('Error pausing job:', error))`).R(
								b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
                xmlns="http://www.w3.org/2000/svg"
                style="vertical-align: middle;">
                <rect x="4" y="4" width="4" height="12" rx="1" fill="currentColor"/>
                <rect x="12" y="4" width="4" height="12" rx="1" fill="currentColor"/>
                </svg>`),
							),
							// Start Now button
							b.AClass("btn btn-primary", "data-job-id", job.JobID, "title", "Start Now", "onClick",
								`fetch('/jobs/run-now/' + this.getAttribute('data-job-id'), {method: 'POST'})
                .then(response => {
                    if (response.ok) return response.json();
                    throw new Error('Network response was not ok');
                })
                .then(data => console.log('Job triggered:', data))
                .catch(error => console.error('Error triggering job:', error))`).R(
								b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
				xmlns="http://www.w3.org/2000/svg"
				style="vertical-align: middle;">
				<polygon points="3,4 11,10 3,16" fill="currentColor"/>
				<rect x="13" y="4" width="3" height="12" fill="currentColor"/>
				</svg>`),
							),
						),
					)

				} else { // run level things
					b.Td().F("%d", job.ResultId)
					b.TdClass("timestamp").T(job.StartTime.Format("2006-01-02 15:04 MST"))
					b.Td().F("%0.1f ms", float64(job.Duration.Microseconds())/1000)
					b.Td().T(job.ResultStatus)
					b.Td().T(job.ErrorMsg)
					b.Td().T("")
				}
			}),
		)
	})
	return
}
