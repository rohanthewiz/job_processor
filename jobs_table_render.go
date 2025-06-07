package main

import (
	_ "embed"
	"job_processor/jobpro"
	"job_processor/util"
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
			// Add Chart.js for mini charts
			b.T(`<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>`),
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
					// Add tooltip to frequency column
					tooltip := ""
					// Use ScheduleType to determine job type
					if strings.ToLower(job.ScheduleType) == "onetime" {
						if !job.NextRunTime.IsZero() {
							tooltip = util.FormatDurationUntil(job.NextRunTime)
						}
					} else if strings.ToLower(job.ScheduleType) == "periodic" && job.FreqType != "" {
						// It's a periodic job with a cron expression
						tooltip = util.ParseCronToEnglish(job.FreqType)
					}

					if tooltip != "" {
						b.TdClass("cron tooltip", "title", tooltip).T(job.FreqType)
					} else {
						b.TdClass("cron").T(job.FreqType)
					}

					statusClass := "badge badge-inactive"
					switch strings.ToLower(job.JobStatus) {
					case "running":
						statusClass = "badge badge-active"
					case "scheduled":
						statusClass = "badge badge-scheduled"
					case "paused":
						statusClass = "badge badge-pending"
					case "pending":
						statusClass = "badge badge-pending"
					case "complete":
						statusClass = "badge badge-complete"
					case "cancelled":
						statusClass = "badge badge-cancelled"
					case "stopped":
						statusClass = "badge badge-stopped"
					case "failed", "error":
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
					// For periodic jobs, show a mini chart in the blank cells area
					if strings.ToLower(job.ScheduleType) == "periodic" {
						// Merge the 5 blank cells (RunID to Error) into one for the chart
						b.Td("colspan", "5").R(
							b.DivClass("chart-container", "style", "height: 60px; width: 100%; position: relative;").R(
								b.Canvas("id", "chart-"+job.JobID, "style", "max-height: 60px;").T(""),
								// Script to fetch and render chart data
								b.Script().T(`
								(function() {
									const chartId = 'chart-`+job.JobID+`';
									const canvas = document.getElementById(chartId);
									if (!canvas) return;
									
									// Fetch job history
									fetch('/jobs/history/`+job.JobID+`')
										.then(response => response.json())
										.then(data => {
											if (!data || data.length === 0) {
												canvas.style.display = 'none';
												return;
											}
											
											// Prepare chart data
											const labels = data.slice().reverse().map((_, idx) => idx + 1);
											const durations = data.slice().reverse().map(d => d.Duration / 1000000); // Convert to ms
											const colors = data.slice().reverse().map(d => 
												d.Status === 'complete' ? '#4ade80' : '#ef4444'
											);
											
											// Create chart
											new Chart(canvas, {
												type: 'line',
												data: {
													labels: labels,
													datasets: [{
														data: durations,
														fill: true,
														backgroundColor: 'rgba(74, 222, 128, 0.2)',
														borderColor: '#4ade80',
														borderWidth: 2,
														pointBackgroundColor: colors,
														pointBorderColor: colors,
														pointRadius: 4,
														pointHoverRadius: 6,
														tension: 0.3
													}]
												},
												options: {
													responsive: true,
													maintainAspectRatio: false,
													plugins: {
														legend: { display: false },
														tooltip: {
															callbacks: {
																label: function(context) {
																	return context.parsed.y.toFixed(1) + ' ms';
																}
															}
														}
													},
													scales: {
														x: { 
															display: false,
															grid: { display: false }
														},
														y: { 
															display: false,
															grid: { display: false },
															beginAtZero: true
														}
													}
												}
											});
										})
										.catch(error => {
											console.error('Error fetching job history:', error);
											canvas.style.display = 'none';
										});
								})();
								`),
							),
						)
					} else {
						// For one-time jobs, keep the blank cells
						b.Td().T("")
						b.Td().T("")
						b.Td().T("")
						b.Td().T("")
						b.Td().T("")
					}
					// Controls
					b.Td().R(
						b.DivClass("btn-group").R(
							b.Wrap(func() {
								// Render controls based on job type and status
								if strings.ToLower(job.ScheduleType) == "periodic" {
									// Periodic job controls with job status
									renderPeriodicJobControls(b, job.JobID, strings.ToLower(job.JobStatus))
								} else {
									// One-time job controls based on status
									renderOneTimeJobControls(b, job.JobID, strings.ToLower(job.JobStatus))
								}
							}),
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

// renderPeriodicJobControls renders control buttons for periodic jobs
func renderPeriodicJobControls(b *element.Builder, jobID string, status string) {
	// Toggle play/pause button based on status
	if status == "paused" {
		// Play/Resume button
		b.AClass("btn btn-primary", "data-job-id", jobID, "title", "Resume Job", "onClick",
			`fetch('/jobs/resume/' + this.getAttribute('data-job-id'), {method: 'POST'}).then(response => { if (response.ok) return response.json(); throw new Error('Network response was not ok'); }).then(data => console.log('Job resumed:', data)).catch(error => console.error('Error resuming job:', error))`).R(
			b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
			xmlns="http://www.w3.org/2000/svg"
			style="vertical-align: middle;">
			<polygon points="5,4 15,10 5,16" fill="currentColor"/>
			</svg>`),
		)
	} else {
		// Pause button (for running/scheduled states)
		b.AClass("btn btn-primary", "data-job-id", jobID, "title", "Pause Job", "onClick",
			`fetch('/jobs/pause/' + this.getAttribute('data-job-id'), {method: 'POST'}).then(response => { if (response.ok) return response.json(); throw new Error('Network response was not ok'); }).then(data => console.log('Job paused:', data)).catch(error => console.error('Error pausing job:', error))`).R(
			b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
			xmlns="http://www.w3.org/2000/svg"
			style="vertical-align: middle;">
			<rect x="4" y="4" width="4" height="12" rx="1" fill="currentColor"/>
			<rect x="12" y="4" width="4" height="12" rx="1" fill="currentColor"/>
			</svg>`),
		)
	}

	// Run Now button - always available
	b.AClass("btn btn-primary", "data-job-id", jobID, "title", "Run Now", "onClick",
		`fetch('/jobs/run-now/' + this.getAttribute('data-job-id'), {method: 'POST'}).then(response => { if (response.ok) return response.json(); throw new Error('Network response was not ok'); }).then(data => console.log('Job triggered:', data)).catch(error => console.error('Error triggering job:', error))`).R(
		b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
		xmlns="http://www.w3.org/2000/svg"
		style="vertical-align: middle;">
		<polygon points="3,4 11,10 3,16" fill="currentColor"/>
		<rect x="13" y="4" width="3" height="12" fill="currentColor"/>
		</svg>`),
	)
}

// renderOneTimeJobControls renders control buttons for one-time jobs based on their status
func renderOneTimeJobControls(b *element.Builder, jobID string, status string) {
	switch status {
	case "created", "cancelled":
		// Start button
		b.AClass("btn btn-primary", "data-job-id", jobID, "title", "Start Job", "onClick",
			`fetch('/jobs/start/' + this.getAttribute('data-job-id'), {method: 'POST'}).then(response => { if (response.ok) return response.json(); throw new Error('Network response was not ok'); }).then(data => console.log('Job started:', data)).catch(error => console.error('Error starting job:', error))`).R(
			b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
			xmlns="http://www.w3.org/2000/svg"
			style="vertical-align: middle;">
			<polygon points="5,4 15,10 5,16" fill="currentColor"/>
			</svg>`),
		)

	case "scheduled":
		// Run Now button
		b.AClass("btn btn-primary", "data-job-id", jobID, "title", "Run Now", "onClick",
			`fetch('/jobs/run-now/' + this.getAttribute('data-job-id'), {method: 'POST'}).then(response => { if (response.ok) return response.json(); throw new Error('Network response was not ok'); }).then(data => console.log('Job triggered:', data)).catch(error => console.error('Error triggering job:', error))`).R(
			b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
			xmlns="http://www.w3.org/2000/svg"
			style="vertical-align: middle;">
			<polygon points="3,4 11,10 3,16" fill="currentColor"/>
			<rect x="13" y="4" width="3" height="12" fill="currentColor"/>
			</svg>`),
		)

		// Reschedule button
		b.AClass("btn btn-secondary", "data-job-id", jobID, "title", "Reschedule Job", "onClick",
			`const newSchedule = prompt('Enter new schedule (e.g., &quot;in 5m&quot;, &quot;2024-12-25 15:00:00 PST&quot;):'); if (newSchedule) { fetch('/jobs/reschedule/' + this.getAttribute('data-job-id'), { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({schedule: newSchedule}) }).then(response => { if (response.ok) return response.json(); throw new Error('Network response was not ok'); }).then(data => console.log('Job rescheduled:', data)).catch(error => console.error('Error rescheduling job:', error)); }`).R(
			b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
			xmlns="http://www.w3.org/2000/svg"
			style="vertical-align: middle;">
			<path d="M10 5V10L13 13" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
			<path d="M17 10C17 13.866 13.866 17 10 17C6.134 17 3 13.866 3 10C3 6.134 6.134 3 10 3C11.5 3 12.9 3.4 14.1 4.1" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
			</svg>`),
		)

		// Cancel button
		b.AClass("btn btn-danger", "data-job-id", jobID, "title", "Cancel Job", "onClick",
			`if (confirm('Are you sure you want to cancel this scheduled job?')) { fetch('/jobs/stop/' + this.getAttribute('data-job-id'), {method: 'POST'}).then(response => { if (response.ok) return response.json(); throw new Error('Network response was not ok'); }).then(data => console.log('Job cancelled:', data)).catch(error => console.error('Error cancelling job:', error)); }`).R(
			b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
			xmlns="http://www.w3.org/2000/svg"
			style="vertical-align: middle;">
			<path d="M6 6L14 14M14 6L6 14" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
			</svg>`),
		)

	case "running":
		// Stop button
		b.AClass("btn btn-danger", "data-job-id", jobID, "title", "Stop Job", "onClick",
			`if (confirm('Are you sure you want to stop this running job?')) { fetch('/jobs/stop/' + this.getAttribute('data-job-id'), {method: 'POST'}).then(response => { if (response.ok) return response.json(); throw new Error('Network response was not ok'); }).then(data => console.log('Job stopped:', data)).catch(error => console.error('Error stopping job:', error)); }`).R(
			b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
			xmlns="http://www.w3.org/2000/svg"
			style="vertical-align: middle;">
			<rect x="4" y="4" width="12" height="12" rx="1" fill="currentColor"/>
			</svg>`),
		)

	case "complete", "failed", "stopped":
		// Retry button
		b.AClass("btn btn-primary", "data-job-id", jobID, "title", "Retry Job", "onClick",
			`fetch('/jobs/run-now/' + this.getAttribute('data-job-id'), {method: 'POST'}).then(response => { if (response.ok) return response.json(); throw new Error('Network response was not ok'); }).then(data => console.log('Job retried:', data)).catch(error => console.error('Error retrying job:', error))`).R(
			b.T(`<svg width="20" height="20" viewBox="0 0 20 20" fill="none"
			xmlns="http://www.w3.org/2000/svg"
			style="vertical-align: middle;">
			<path d="M4 10C4 6.686 6.686 4 10 4C12.2 4 14.1 5.2 15.1 7" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
			<path d="M16 10C16 13.314 13.314 16 10 16C7.8 16 5.9 14.8 4.9 13" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
			<polyline points="15 3 15 7 11 7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
			<polyline points="5 17 5 13 9 13" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
			</svg>`),
		)
	}
}
