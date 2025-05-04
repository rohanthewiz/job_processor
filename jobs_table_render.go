package main

import (
	_ "embed"
	"fmt"
	"job_processor/jobpro"

	"github.com/rohanthewiz/element"
)

//go:embed assets/table_styles.css
var tableStyles string

func renderJobsTable(jobs []jobpro.DisplayResults) string {
	b := element.NewBuilder()
	cols := []string{"Job", "ID", "Freq", "Status", "Created", "Updated",
		"Run ID", "Run Start", "Run Duration", "Run Status", "Run Error Msg"}

	b.Html().R(
		b.Head().R(
			b.Title().R("Jobs Management"),
			// Add meta tag for responsive design
			b.T(`<meta name="viewport" content="width=device-width, initial-scale=1.0">`),
			// Add modern CSS styling
			b.Style().T(tableStyles),
		),
		b.Body().R(
			b.Div("class", "container").R(
				b.H1().T("Job Management"),
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
							element.ForEach(jobs, func(job jobpro.DisplayResults) {
								b.Tr().R(
									b.Td().T(job.JobName),
									b.Td().T(job.JobID),
									b.Td().T(job.FreqType),
									b.Wrap(func() {
										statusClass := "badge badge-inactive"
										switch job.JobState {
										case "active":
											statusClass = "badge badge-active"
										case "pending":
											statusClass = "badge badge-pending"
										case "error":
											statusClass = "badge badge-error"
										}
										b.Td().R(
											b.Span("class", statusClass).T(job.JobState),
										)
									}),
									b.Td("class", "timestamp").T(job.CreatedAt.Format("2006-01-02 15:04 MST")),
									b.Td("class", "timestamp").T(job.UpdatedAt.Format("2006-01-02 15:04 MST")),
									b.Td().T(fmt.Sprintf("%d", job.ResultId)),
									b.Td("class", "timestamp").T(job.StartTime.Format("2006-01-02 15:04 MST")),
									b.Td().T(fmt.Sprintf("%s", job.Duration)),
									b.Td().T(job.ResultStatus),
									b.Td().T(job.ErrorMsg),
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
