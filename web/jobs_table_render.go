package web

import (
	_ "embed"
	"fmt"
	"job_processor/jobpro"
	"job_processor/util"
	"strings"

	"github.com/rohanthewiz/element"
)

//go:embed assets/table_styles.css
var tableStyles string

//go:embed assets/table_rows.js
var tableRows string

//go:embed assets/periodic_job_row.js
var periodicJobRow string

//go:embed assets/time_tooltip.js
var timeTooltip string

const jobEvent = "job-update"

const barChartEmoji = `<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" style="vertical-align: middle;"><rect x="1" y="8" width="2" height="6" fill="currentColor"/><rect x="4" y="4" width="2" height="10" fill="currentColor"/><rect x="7" y="6" width="2" height="8" fill="currentColor"/><rect x="10" y="2" width="2" height="12" fill="currentColor"/><rect x="13" y="10" width="2" height="4" fill="currentColor"/></svg>`

const stopWatchEmoji = `<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" style="vertical-align: middle;"><circle cx="8" cy="9" r="6" stroke="currentColor" stroke-width="1.5" fill="none"/><path d="M8 6v3l2 2" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><rect x="6" y="1" width="4" height="2" rx="1" fill="currentColor"/><circle cx="8" cy="9" r="1" fill="currentColor"/></svg>`

// renderJobsTable renders the full jobs table page
func renderJobsTable(jobs []jobpro.JobRun, resultCounts map[string]int) string {
	b := element.NewBuilder()
	cols := []string{"Job", "Id", "Freq", "Status", "Created", "Updated",
		"Run&nbsp;Id", "Run Start", "Duration", "Status", "Error", "Controls"}

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
			// Add time tooltip functionality
			b.Script().T(timeTooltip),
		),
		b.Body().R(
			// Add SSE source connection to the body
			b.DivClass("container").R(
				b.H1Class("table-title").T("JOBS"),
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

							renderJobsTableRows(b, jobs, resultCounts),
						),
					),
				),
			),
		),
	)

	return b.String()
}

// renderJobsTableRows renders just the table rows - for HTMX updates
func renderJobsTableRows(b *element.Builder, jobs []jobpro.JobRun, resultCounts map[string]int) (x any) {
	// Add JavaScript for expand/collapse functionality and load more
	b.Script().T(tableRows)

	// Track results per job
	currentJobID := ""
	resultCount := 0
	displayedResults := make(map[string]int)
	lastJobID := ""

	for _, job := range jobs {
		// Determine if this is a main job row or a result row
		isMainRow := job.ResultId == 0

		// If we've moved to a new job and the previous job has more results
		if isMainRow && lastJobID != "" && lastJobID != job.JobID {
			if total, exists := resultCounts[lastJobID]; exists && displayedResults[lastJobID] < total {
				// Add load more button for the previous job
				remaining := total - displayedResults[lastJobID]
				loadCount := 10
				if remaining < loadCount {
					loadCount = remaining
				}

				b.TrClass(fmt.Sprintf("load-more-row job-%s", lastJobID),
					"data-job-id", lastJobID,
					"style", "display: none;").R(
					b.Td("colspan", "12", "style", "text-align: center; padding: 10px;").R(
						b.ButtonClass("btn btn-secondary load-more-btn",
							"data-job-id", lastJobID,
							"data-offset", fmt.Sprintf("%d", displayedResults[lastJobID]),
							"data-total", fmt.Sprintf("%d", total),
							"onclick", "loadMoreResults(this)").F(
							"(%d / %d) <b>load more</b>",
							loadCount,
							total,
						),
					),
				)
			}
		}

		if isMainRow {
			currentJobID = job.JobID
			resultCount = 0
			lastJobID = job.JobID
		} else {
			resultCount++
			displayedResults[currentJobID] = resultCount
		}
		rowClass := ""
		if isMainRow {
			rowClass = "job-main-row"
		} else {
			rowClass = "job-result-row"
		}

		b.Tr("class", rowClass, "data-job-id", job.JobID, "style", func() string {
			if !isMainRow {
				// Check if this job should be hidden initially
				return "display: none;"
			}
			return ""
		}()).R(
			b.Td().R(
				b.Wrap(func() {
					if isMainRow {
						// For main rows, use flex container with toggle button
						b.Div("style", "display: flex; align-items: center; gap: 0.5rem;").R(
							b.Span("class", "toggle-btn", "data-job-id", job.JobID,
								"onclick", "toggleJobResults('"+job.JobID+"')",
								"style", "cursor: pointer; font-size: 0.8rem; user-select: none; flex-shrink: 0;").T("&#9658;"),
							b.Span("style", "flex-grow: 1;").T(job.JobName),
							// Add result count indicator for periodic jobs
							b.Wrap(func() {
								if strings.ToLower(job.ScheduleType) == "periodic" {
									if total, exists := resultCounts[job.JobID]; exists && total > 0 {
										displayed := displayedResults[job.JobID]
										if displayed == 0 {
											displayed = 10 // We're showing up to 10 by default
											if total < displayed {
												displayed = total
											}
										}
										b.SpanClass("result-count",
											"style", "font-size: 0.8rem; color: #666; white-space: nowrap;").
											F("(%d of %d)", displayed, total)
									}
								}
							}),
						)
					} else {
						// For result rows, just show the name without toggle
						b.T(job.JobName)
					}
				}),
			),
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
					b.TdClass("timestamp").T(job.CreatedAt.UTC().Format("2006-01-02 15:04 MST"))
					b.TdClass("timestamp").T(job.UpdatedAt.UTC().Format("2006-01-02 15:04 MST"))

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
						b.Td("colspan", "5", "style", "position: relative; padding: 0.75rem 1rem;").R(
							b.DivClass("chart-container", "style", "height: 60px; width: 100%; max-width: 450px; position: relative; display: flex; align-items: center; gap: 0.5rem; overflow: hidden;").R(
								b.DivClass("success-rate-container", "id", "success-rate-"+job.JobID,
									"style", "font-size: 0.9rem; white-space: nowrap; width: 70px; flex-shrink: 0; text-align: center;").T(""),
								b.Div("style", "flex: 1; overflow: hidden; position: relative;").R(
									b.Canvas("id", "chart-"+job.JobID, "style", "display: block; max-height: 60px;").T(""),
								),
								// Script to fetch and render chart data
								b.Script().T(`(function(jobID) {`+periodicJobRow+`})('`+job.JobID+`');`),
							),
						)
					} else {
						// For one-time jobs, show a summary in the blank cells area
						b.Td("colspan", "5").R(
							b.DivClass("summary-container", "id", "summary-"+job.JobID).R(
								// Script to fetch and render summary data
								b.Script().T(`
								(function() {
									const summaryId = 'summary-` + job.JobID + `';
									const container = document.getElementById(summaryId);
									if (!container) return;
									
									// Fetch job history
									fetch('/jobs/history/` + job.JobID + `')
										.then(response => response.json())
										.then(data => {
											if (!data || data.length === 0) {
												container.innerHTML = '<div class="summary-empty">No runs yet</div>';
												return;
											}
											
											// Calculate statistics
											const totalRuns = data.length;
											const successfulRuns = data.filter(d => d.Status === 'complete').length;
											const successRate = Math.round((successfulRuns / totalRuns) * 100);
											const durations = data.map(d => d.Duration / 1000000); // Convert to ms
											const avgDuration = durations.reduce((a, b) => a + b, 0) / durations.length;
											
											// Get last run info
											const lastRun = data[0]; // Most recent
											const lastRunTime = new Date(lastRun.StartTime);
											const timeSince = getTimeSince(lastRunTime);
											const lastRunStatus = lastRun.Status === 'complete' ? '&#10003;' : '&#10007;';
											const lastRunClass = lastRun.Status === 'complete' ? 'success' : 'error';
											
											// Create summary HTML
											const summaryHTML = 
												'<div class="summary-stats" style="display: flex; flex-direction: row; gap: 1.5rem; width: 100%;">' +
												'<div class="stat" style="display: flex; align-items: center; gap: 0.4rem;">' +
												'<span class="stat-icon">` + barChartEmoji + `</span>' +
												'<span class="stat-value">' + totalRuns + '</span>' +
												'<span class="stat-label">' + (totalRuns === 1 ? 'run' : 'runs') + '</span>' +
												'</div>' +
												'<div class="stat" style="display: flex; align-items: center; gap: 0.4rem;">' +
												'<span class="stat-icon ' + (successRate >= 80 ? 'success' : successRate >= 50 ? 'warning' : 'error') + '">&#10003;</span>' +
												'<span class="stat-value">' + successRate + '%</span>' +
												'<span class="stat-label">success</span>' +
												'</div>' +
												'<div class="stat" style="display: flex; align-items: center; gap: 0.4rem;">' +
												'<span class="stat-icon">` + stopWatchEmoji + `</span>' +
												'<span class="stat-value">' + avgDuration.toFixed(1) + 'ms</span>' +
												'<span class="stat-label">avg</span>' +
												'</div>' +
												'<div class="stat" style="display: flex; align-items: center; gap: 0.4rem;">' +
												'<span class="stat-icon ' + lastRunClass + '">' + lastRunStatus + '</span>' +
												'<span class="stat-value">' + timeSince + '</span>' +
												'<span class="stat-label">ago</span>' +
												'</div>' +
												'</div>';
											
											container.innerHTML = summaryHTML;
										})
										.catch(error => {
											console.error('Error fetching job history:', error);
											container.innerHTML = '<div class="summary-error">Failed to load stats</div>';
										});
									
									// Helper function to calculate time since
									function getTimeSince(date) {
										const seconds = Math.floor((new Date() - date) / 1000);
										if (seconds < 60) return seconds + 's';
										const minutes = Math.floor(seconds / 60);
										if (minutes < 60) return minutes + 'm';
										const hours = Math.floor(minutes / 60);
										if (hours < 24) return hours + 'h';
										const days = Math.floor(hours / 24);
										return days + 'd';
									}
								})();
								`),
							),
						)
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
					b.Td().F("#%d", job.RunNumber)
					b.TdClass("timestamp").T(job.StartTime.UTC().Format("2006-01-02 15:04 MST"))
					b.Td().F("%0.1f ms", float64(job.Duration.Microseconds())/1000)
					b.Td().T(job.ResultStatus)
					b.Td().T(job.ErrorMsg)
					b.Td().T("")
				}
			}),
		)
	}

	// Add load more button for the last job if needed
	if lastJobID != "" {
		if total, exists := resultCounts[lastJobID]; exists && displayedResults[lastJobID] < total {
			remaining := total - displayedResults[lastJobID]
			loadCount := 10
			if remaining < loadCount {
				loadCount = remaining
			}

			b.Tr("class", fmt.Sprintf("load-more-row job-%s", lastJobID),
				"data-job-id", lastJobID,
				"style", "display: none;").R(
				b.Td("colspan", "6", "style", "text-align: center; padding: 10px;").R(
					b.ButtonClass("btn btn-secondary load-more-btn",
						"data-job-id", lastJobID,
						"data-offset", fmt.Sprintf("%d", displayedResults[lastJobID]),
						"data-total", fmt.Sprintf("%d", total),
						"onclick", "loadMoreResults(this)").F(
						"(%d / %d) <b>load more</b>",
						// loadCount,
						displayedResults[lastJobID],
						total,
					),
				),
			)
		}
	}

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
