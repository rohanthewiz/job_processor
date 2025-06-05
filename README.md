# Go Job Processor

A lightweight, flexible job scheduling and processing system written in Go.

## Features

- **Comprehensive Job Management**: Create, Start, Stop, Pause, Resume, and Delete operations for jobs
- **Flexible Scheduling**: Support for both one-time and periodic jobs using cron syntax
- **Persistent Storage**: Jobs and their execution history stored in DuckDB
- **Runtime Metrics**: Detailed statistics for each job execution
- **Graceful Shutdown**: Clean handling of shutdown signals
- **Error Handling**: Structured error system

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

### Basic Usage
See `main.go` for a complete example.

```go

### Registering Jobs

- Periodic jobs are registered with a cron schedule
- Onetime jobs are registered with a time string of when to run

Register jobs using `jobpro.RegisterJob()` with a `JobConfig`:

```go
// Register a periodic job
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

// Register a one-time job
jobpro.RegisterJob(jobpro.JobConfig{
	ID:         "onetimeJob1",
	Name:       "Onetime Job 1",
	IsPeriodic: false,
	Schedule:   time.Now().Add(30 * time.Second).Format(time.RFC3339),
	JobFunction: func() error {
		fmt.Println("One time job doing work")
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
