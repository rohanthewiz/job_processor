# Manual Start Job Tests

This file documents the tests for Manual Start job functionality in the job processor.

## Running the Tests

To run all manual job related tests:

```bash
go test -v ./jobpro -run "TestManual|TestJobStatus"
```

To run a specific test:

```bash
go test -v ./jobpro -run TestManualStartJob
```

## Test Coverage

### TestManualStartJob
Tests the core functionality of manual start jobs:
- **Manual jobs don't auto-start**: Jobs with `AutoStart=false` and no schedule remain in created state
- **Auto-start jobs execute immediately**: Jobs with `AutoStart=true` run right away
- **Scheduled jobs respect their schedule**: Jobs with schedules run at the specified time
- **Manual start works**: Calling `StartJob()` on a manual job executes it immediately
- **Multiple triggers work**: Manual jobs can be triggered multiple times using `TriggerJobNow()`

### TestManualJobWithSchedule
Tests manual jobs that have a schedule:
- Manual jobs with future schedules don't run until started
- When started, they wait for their scheduled time (not immediate)
- The schedule is respected even for manual start jobs

### TestJobStatusTransitions
Tests job status changes for manual jobs:
- Initial status is `StatusCreated`
- Changes to `StatusRunning` when started
- Changes to `StatusComplete` when finished

### TestManualJobPersistence
Tests persistence across system restarts:
- Manual jobs maintain their status after restart
- They don't auto-start after system restart
- Job definitions are preserved in the database

## Key Behaviors Verified

1. **Manual Start Jobs** (AutoStart=false, no schedule):
   - Don't start automatically when registered
   - Execute immediately when `StartJob()` is called
   - Can be triggered multiple times

2. **Auto-Start Jobs** (AutoStart=true):
   - Start automatically when registered
   - Execute based on their schedule (or immediately if no schedule)

3. **Scheduled Manual Jobs** (AutoStart=false, with schedule):
   - Don't start automatically
   - When started manually, wait for scheduled time
   - Don't execute immediately even when started

## Implementation Details

The fix involved two main changes:

1. **register.go**: Simplified the auto-start condition to only check `AutoStart` flag
2. **job_manager.go**: Added logic to execute manual jobs (empty schedule) immediately when started

This ensures manual jobs behave as expected - they wait for user action before running.