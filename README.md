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

- **Job Interface**: Any struct that implements the `Run()` method can be a job
- **JobManager**: Handles job lifecycle and scheduling
- **JobStore**: Persists job definitions and execution results
- **Stats**: Captures runtime metrics for job executions

## Getting Started

### Prerequisites

- Go 1.22+
- DuckDB

### Installation

```bash
go get github.com/rohanthewiz/jobprocessor
```

### Basic Usage
See `main.go` for a complete example.

```go

### Creating Custom Jobs - This is the basic idea but may not be complete. See the example in `main.go`.

Implement the `Job` interface to create custom jobs:

```go
type MyCustomJob struct {
	id   string
	name string
}

func (j *MyCustomJob) ID() string {
	return j.id
}

func (j *MyCustomJob) Name() string {
	return j.name
}

func (j *MyCustomJob) Type() jobprocessor.ScheduleType {
	return jobprocessor.SchedulePeriodic
}

func (j *MyCustomJob) Run(ctx context.Context) (jobprocessor.Stats, error) {
	stats := jobprocessor.Stats{
		StartTimeUTC: time.Now().UTC(),
	}
	
	// Your custom job logic here
	// ...
	
	stats.Duration = time.Since(stats.StartTimeUTC)
	stats.SuccessMsg = "Job completed successfully"
	return stats, nil
}
```

## Scheduling Jobs

### One-Time Jobs

For one-time jobs, specify the execution time in RFC3339 format:

```go
jobID, err := manager.CreateJob(job, time.Now().Add(1*time.Hour).Format(time.RFC3339))
```

### Periodic Jobs

For periodic jobs, use standard cron syntax:

```go
// Run every 15 minutes
jobID, err := manager.CreateJob(job, "0 */15 * * * *")

// Run at 2:30am every day
jobID, err := manager.CreateJob(job, "0 30 2 * * *")
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
