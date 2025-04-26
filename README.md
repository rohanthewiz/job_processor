# Go Job Processor

A lightweight, flexible job scheduling and processing system written in Go.

## Features

- **Comprehensive Job Management**: Create, Start, Stop, Pause, Resume, and Delete operations for jobs
- **Flexible Scheduling**: Support for both one-time and periodic jobs using cron syntax
- **Persistent Storage**: Jobs and their execution history stored in DuckDB
- **Runtime Metrics**: Detailed statistics for each job execution
- **Graceful Shutdown**: Clean handling of shutdown signals
- **Error Handling**: Structured error system with stack traces

## Core Components

- **Job Interface**: Any struct that implements the `Run()` method can be a job
- **JobManager**: Handles job lifecycle and scheduling
- **JobStore**: Persists job definitions and execution results
- **Stats**: Captures runtime metrics for job executions

## Getting Started

### Prerequisites

- Go 1.18+
- DuckDB

### Installation

```bash
go get github.com/your-username/jobpro
```

### Basic Usage

```go
package main

import (
	"log"
	"time"

	"github.com/your-username/jobprocessor"
)

func main() {
	// Initialize the DuckDB store
	store, err := jobprocessor.NewDuckDBStore("jobs.duckdb")
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	// Create the job manager
	manager := jobprocessor.NewJobManager(store)

	// Create a simple dummy job that runs once
	job := jobprocessor.NewDummyJob(
		"job-1",
		"My First Job",
		jobprocessor.ScheduleOneTime,
		5*time.Second,
		0.9, // 90% chance of success
	)

	// Add the job to the manager
	jobID, err := manager.CreateJob(job, time.Now().Add(1*time.Minute).Format(time.RFC3339))
	if err != nil {
		log.Fatalf("Failed to create job: %v", err)
	}

	// Start the job
	if err := manager.StartJob(jobID); err != nil {
		log.Fatalf("Failed to start job: %v", err)
	}

	// Wait for completion
	time.Sleep(2 * time.Minute)
}
```

### Creating Custom Jobs

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
jobID, err := manager.CreateJob(job, "*/15 * * * *")

// Run at 2:30am every day
jobID, err := manager.CreateJob(job, "30 2 * * *")
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

## Error Handling

The package includes a structured error handling system:

```go
import "github.com/your-username/jobpro/serr"

// Create a new error
err := serr.New("something went wrong")

// Wrap an existing error
wrappedErr := serr.Wrap(err, "while processing file")

// Format and wrap
formattedErr := serr.Wrapf(err, "while processing file %s", filename)

// Get full error with stack trace
fullError := wrappedErr.FullError()
```

## Signal Handling & Graceful Shutdown

The main application automatically handles SIGINT and SIGTERM signals, allowing for graceful shutdown of running jobs.

## License

MIT
