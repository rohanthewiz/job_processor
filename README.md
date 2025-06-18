# Go Job Processor

A lightweight, flexible job scheduling and processing system written in Go.

## Features

- **Comprehensive Job Management**: Create, Start, Stop, Pause, Resume, and Delete operations for jobs
- **Flexible Scheduling**: Support for both one-time and periodic jobs using cron syntax
- **Persistent Storage**: Jobs and their execution history stored in DuckDB
- **Runtime Metrics**: Detailed statistics for each job execution
- **Graceful Shutdown**: Clean handling of shutdown signals
- **Error Handling**: Structured error system

![Screenshot 2025-06-17 at 6 36 08 PM](https://github.com/user-attachments/assets/0b5d7d04-0755-4463-b1fe-3ccd56a513a2)

## Core Components

- **JobManager**: Handles job lifecycle and scheduling
- **JobStore**: Persists job definitions and execution results
- **Stats**: Captures runtime metrics for job executions

## Getting Started

### Prerequisites

- Go 1.22+

### Installation

```bash
go get github.com/rohanthewiz/jobprocessor
```

### Usage
**See `main.go` for a complete example**.

### Registering Jobs

Register jobs using `jobpro.RegisterJob()` with a `JobConfig`:

#### Periodic Jobs
Periodic jobs use cron syntax for scheduling:

```go
jobpro.RegisterJob(jobpro.JobConfig{
	ID:         "periodicJob1",
	Name:       "Periodic Job 1",
	IsPeriodic: true,
	Schedule:   "*/15 * * * * *", // Every 15 seconds
	JobFunction: func() error {
		fmt.Println("Periodic job doing work")
		return nil
	},
})
```

#### One-Time Jobs
One-time jobs support multiple time format options:

**Relative Time Formats:**
- `"in 30m"` - 30 minutes from now
- `"+1h"` - 1 hour from now
- `"5m"` - 5 minutes from now
- `"in 2h30m"` - 2.5 hours from now

**Absolute Time Formats:**
- RFC3339: `"2024-01-15T14:30:00-08:00"`
- With timezone abbreviation: `"2024-01-15 14:30:00 PST"`
- With IANA timezone: `"2024-01-15 14:30:00 America/Los_Angeles"`
- US format: `"01/15/2024 2:30 PM EST"`
- Human readable: `"Jan 15, 2024 3:04 PM MST"`

```go
// Using relative time
jobpro.RegisterJob(jobpro.JobConfig{
	ID:         "onetimeJob1",
	Name:       "Onetime Job 1",
	IsPeriodic: false,
	Schedule:   "in 30s", // Runs 30 seconds from now
	JobFunction: func() error {
		fmt.Println("One time job doing work")
		return nil
	},
})

// Using absolute time with timezone
jobpro.RegisterJob(jobpro.JobConfig{
	ID:         "manualJob",
	Name:       "Onetime Job 2",
	IsPeriodic: false,
	Schedule:   "2024-12-25 09:00:00 EST",
	JobFunction: func() error {
		fmt.Println("Christmas morning job")
		return nil
	},
})
```

## Job Lifecycle Operations

```go
// Pause a job
err := manager.PauseJob(jobID)

// Resume a paused job
err := manager.ResumeJob(jobID)

// Stop a job completely
err := manager.StopJob(jobID)

// Delete a job
err := manager.DeleteJob(jobID)

// Get current job status
status, err := manager.GetJobStatus(jobID)
```

## Signal Handling & Graceful Shutdown

The main application automatically handles SIGINT and SIGTERM signals, allowing for graceful shutdown of running jobs.

## License

MIT
