# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build & Run
```bash
go build                    # Build the binary
go run main.go             # Run directly
./job_processor            # Run the built binary
```

### Testing
```bash
go test ./...              # Run all tests
go test -v ./...           # Verbose test output
go test -race ./...        # Test with race detector
go test ./jobpro/...       # Test specific package
```

### Code Quality
```bash
go fmt ./...               # Format code
go vet ./...               # Run static analysis
go mod tidy                # Clean up dependencies
```

## Architecture Overview

This is a job scheduling and processing system that manages both one-time and periodic (cron-based) jobs with persistent storage and real-time monitoring.

### Core Components

1. **Job Interface** (`jobpro/job_core.go:10`): Central abstraction requiring `Run()`, `ID()`, `Name()`, and `Type()` methods. Jobs execute with context and return stats including duration and success/error status.

2. **Job Manager** (`jobpro/job_manager.go`): Orchestrates job lifecycle using `robfig/cron` for scheduling. Maintains concurrent execution with goroutines, context cancellation, and thread-safe operations for Create, Start, Stop, Pause, Resume, Delete, and Trigger.

3. **DuckDB Storage** (`jobpro/duckdb_store.go`): Persists job definitions and execution history. Implements automatic cleanup of results older than 1 week. The `jobs` table stores job configurations while `job_results` tracks execution history.

4. **HTTP API** (`main.go:29-47`): RESTful endpoints under `/jobs/` for management operations. Server-Sent Events provide real-time status updates. HTML table rendering (`jobs_table_render.go`) displays job states visually.

5. **Job Registration** (`jobpro/register.go`): Jobs are registered at startup using `RegisterJob()` with `JobConfig` structs specifying type (OneTime/Periodic), cron schedule, and implementation.

### Key Patterns

- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals, cancels running jobs cleanly
- **Pub/Sub Updates**: Real-time notifications via internal broker for job state changes
- **Error Handling**: Uses `rohanthewiz/serr` for wrapped errors with context
- **Concurrent Safety**: Mutexes protect shared state in job manager

### Adding New Jobs

1. Implement the `Job` interface from `jobpro/job_core.go`
2. Register using `jobpro.RegisterJob()` with appropriate `JobConfig`
3. For periodic jobs, provide valid cron expression (no additional period prefix)
4. One-time jobs execute immediately on creation unless paused

### Database Schema

Jobs table stores configurations, job_results tracks executions with timestamps, durations, and success/error messages. Old results are automatically cleaned up by the cleanup job.