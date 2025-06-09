package jobpro

import (
	"database/sql"
	"fmt"
	"job_processor/util"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rohanthewiz/serr"
)

// DuckDBStore implements JobStore using DuckDB
type DuckDBStore struct {
	db *sql.DB
}

// NewDuckDBStore creates a new DuckDB-backed job store
func NewDuckDBStore(dbPath string) (*DuckDBStore, error) {
	fmt.Printf("Job Store DB Path: %s\n",
		util.Tern(dbPath == "", "(in-memory)", dbPath))

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}

	store := &DuckDBStore{db: db}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// initialize creates the necessary tables if they don't exist
func (s *DuckDBStore) initialize() error {
	// Create jobs table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			job_id VARCHAR PRIMARY KEY,
			job_name VARCHAR NOT NULL,
			schedule_type VARCHAR NOT NULL,
			schedule VARCHAR,
			next_run_time TIMESTAMP,
			status VARCHAR NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create jobs table: %w", err)
	}

	// Create job results table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS job_results (
			result_id INTEGER PRIMARY KEY,
			job_id VARCHAR NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL,
			duration_micro BIGINT NOT NULL,
			status VARCHAR NOT NULL,
			success_msg VARCHAR,
			error_msg VARCHAR,
			FOREIGN KEY (job_id) REFERENCES jobs(job_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create job_results table: %w", err)
	}

	// Create sequence for job_results result_id
	_, err = s.db.Exec(`
		CREATE SEQUENCE IF NOT EXISTS job_results_id_seq
	`)
	if err != nil {
		return fmt.Errorf("failed to create job_results sequence: %w", err)
	}

	return nil
}

// SaveJob persists a job definition
func (s *DuckDBStore) SaveJob(job JobDef) error {
	_, err := s.db.Exec(`
		INSERT INTO jobs (
			job_id, job_name, schedule_type, schedule, 
			next_run_time, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (job_id) DO UPDATE SET
			job_name = excluded.job_name,
			schedule_type = excluded.schedule_type,
			schedule = excluded.schedule,
			next_run_time = excluded.next_run_time,
			status = excluded.status,
			updated_at = excluded.updated_at
	`,
		job.JobID, job.JobName, job.SchedType, job.Schedule,
		job.NextRunTime, job.Status, job.CreatedAt, job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}
	return nil
}

// GetJob retrieves a job definition by ID
func (s *DuckDBStore) GetJob(id string) (JobDef, error) {
	row := s.db.QueryRow(`
		SELECT job_id, job_name, schedule_type, schedule, 
		       next_run_time, status, created_at, updated_at
		FROM jobs WHERE job_id = ?
	`, id)

	var job JobDef
	err := row.Scan(
		&job.JobID, &job.JobName, &job.SchedType, &job.Schedule,
		&job.NextRunTime, &job.Status, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return JobDef{}, fmt.Errorf("failed to get job: %w", err)
	}
	return job, nil
}

// ListJobs retrieves all job definitions with optional filters
func (s *DuckDBStore) ListJobs(status JobStatus, schedType FreqType) ([]JobDef, error) {
	query := `
		SELECT job_id, job_name, schedule_type, schedule, 
		       next_run_time, status, created_at, updated_at
		FROM jobs
	`
	args := []interface{}{}
	where := []string{}

	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}

	if schedType != "" {
		where = append(where, "schedule_type = ?")
		args = append(args, schedType)
	}

	if len(where) > 0 {
		query += " WHERE " + where[0]
		for i := 1; i < len(where); i++ {
			query += " AND " + where[i]
		}
	}

	query += " ORDER BY next_run_time ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	jobs := []JobDef{}
	for rows.Next() {
		var job JobDef
		err := rows.Scan(
			&job.JobID, &job.JobName, &job.SchedType, &job.Schedule,
			&job.NextRunTime, &job.Status, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// UpdateJobStatus updates the status of a job
func (s *DuckDBStore) UpdateJobStatus(id string, status JobStatus) error {
	_, err := s.db.Exec(`
		UPDATE jobs SET status = ?, updated_at = ? WHERE job_id = ?
	`, status, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return nil
}

// UpdateNextRunTime updates when a job should next run
func (s *DuckDBStore) UpdateNextRunTime(id string, nextRun time.Time) error {
	_, err := s.db.Exec(`
		UPDATE jobs SET next_run_time = ?, updated_at = ? WHERE job_id = ?
	`, nextRun, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update next run time: %w", err)
	}
	return nil
}

// DeleteJob removes a job definition
func (s *DuckDBStore) DeleteJob(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete job results first due to foreign key constraint
	_, err = tx.Exec("DELETE FROM job_results WHERE job_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete job results: %w", err)
	}

	// Delete the job
	_, err = tx.Exec("DELETE FROM jobs WHERE job_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return tx.Commit()
}

// RecordJobResult stores the outcome of a job execution
func (s *DuckDBStore) RecordJobResult(result JobResult) error {
	durationMicro := result.Duration.Microseconds()

	_, err := s.db.Exec(`
		INSERT INTO job_results (
			result_id, job_id, start_time, end_time, duration_micro, 
			status, success_msg, error_msg
		) VALUES (nextval('job_results_id_seq'), ?, ?, ?, ?, ?, ?, ?)
	`,
		result.JobID, result.StartTime, result.EndTime, durationMicro,
		result.Status, result.SuccessMsg, result.ErrorMsg,
	)
	if err != nil {
		return fmt.Errorf("failed to record job result: %w", err)
	}
	return nil
}

// GetJobResults retrieves historical results for a job
func (s *DuckDBStore) GetJobResults(jobID string, limit int) ([]JobResult, error) {
	rows, err := s.db.Query(`
		SELECT job_id, start_time, end_time, duration_micro, 
		       status, success_msg, error_msg
		FROM job_results
		WHERE job_id = ?
		ORDER BY start_time DESC
		LIMIT ?
	`, jobID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get job results: %w", err)
	}
	defer rows.Close()

	results := []JobResult{}
	for rows.Next() {
		var result JobResult
		var durationMicro int64
		err := rows.Scan(
			&result.JobID, &result.StartTime, &result.EndTime, &durationMicro,
			&result.Status, &result.SuccessMsg, &result.ErrorMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result row: %w", err)
		}
		result.Duration = time.Duration(durationMicro) * time.Microsecond
		results = append(results, result)
	}

	return results, nil
}

type JobRun struct {
	JobID        string
	JobName      string
	FreqType     string
	Schedule     string
	NextRunTime  time.Time
	JobStatus    string
	ScheduleType string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ResultId     int64
	StartTime    time.Time
	Duration     time.Duration
	ResultStatus string
	ErrorMsg     string
	RunNumber    int
}

type JobRunDBRow struct {
	JobID        string
	JobName      sql.NullString
	FreqType     sql.NullString
	Schedule     sql.NullString
	NextRunTime  sql.NullTime
	JobStatus    sql.NullString
	ScheduleType sql.NullString
	CreatedAt    sql.NullTime
	UpdatedAt    sql.NullTime
	ResultId     sql.NullInt64
	StartTime    sql.NullTime
	// Duration     time.Duration
	ResultStatus sql.NullString
	ErrorMsg     sql.NullString
	RunNumber    sql.NullInt64
}

// GetJobRunsWithPagination retrieves jobs with limited results per job
func (s *DuckDBStore) GetJobRunsWithPagination(resultsPerJob int) ([]JobRun, map[string]int, error) {
	// Map to store total result count per job
	resultCounts := make(map[string]int)
	
	// Get total count of results per job
	countRows, err := s.db.Query(`
		SELECT job_id, COUNT(*) as total_count 
		FROM job_results 
		GROUP BY job_id
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get result counts: %w", err)
	}
	defer countRows.Close()
	
	for countRows.Next() {
		var jobID string
		var count int
		if err := countRows.Scan(&jobID, &count); err != nil {
			return nil, nil, fmt.Errorf("failed to scan count row: %w", err)
		}
		resultCounts[jobID] = count
	}

	// Build the query with pagination per job
	// First get all job main rows, then union with limited results per job
	query := `
	WITH job_counts AS (
		SELECT job_id, COUNT(*) as total_count
		FROM job_results
		GROUP BY job_id
	),
	job_main_rows AS (
		SELECT j.job_id, j.job_name, 
			   CASE WHEN j.schedule IS NULL OR j.schedule = '' THEN 'one-time' ELSE j.schedule END as frequency,
			   j.schedule, j.next_run_time, j.status, j.schedule_type, j.created_at, j.updated_at,
			   NULL::BIGINT as result_id, NULL::TIMESTAMP as start_time, NULL::BIGINT as duration_micro, 
			   NULL::VARCHAR as result_status, NULL::VARCHAR as error_msg,
			   0 as row_type, NULL::INT as run_number
		FROM jobs j
	),
	ranked_results AS (
		SELECT r.job_id, NULL as job_name, NULL as frequency, NULL as schedule, 
			   NULL::TIMESTAMP as next_run_time, NULL as status, NULL as schedule_type, 
			   j.created_at, NULL::TIMESTAMP as updated_at,
			   r.result_id, r.start_time, r.duration_micro, r.status as result_status, r.error_msg,
			   1 as row_type,
			   ROW_NUMBER() OVER (PARTITION BY r.job_id ORDER BY r.start_time DESC) as rn,
			   (jc.total_count - ROW_NUMBER() OVER (PARTITION BY r.job_id ORDER BY r.start_time DESC) + 1) as run_number
		FROM job_results r
		JOIN jobs j ON r.job_id = j.job_id
		JOIN job_counts jc ON r.job_id = jc.job_id
		WHERE r.job_id IN (SELECT job_id FROM jobs)
	),
	limited_results AS (
		SELECT * FROM ranked_results WHERE rn <= ?
	),
	all_rows AS (
		SELECT * FROM job_main_rows
		UNION ALL
		SELECT job_id, job_name, frequency, schedule, next_run_time, status, 
			   schedule_type, created_at, updated_at, result_id, start_time, 
			   duration_micro, result_status, error_msg, row_type, run_number
		FROM limited_results
	)
	SELECT job_id, job_name, frequency, schedule, next_run_time, status,
		   schedule_type, created_at, updated_at, result_id, start_time, 
		   duration_micro, result_status, error_msg, run_number
	FROM all_rows
	ORDER BY created_at DESC, job_id, row_type, start_time DESC
	`

	rows, err := s.db.Query(query, resultsPerJob)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute paginated query: %w", err)
	}
	defer rows.Close()

	results := make([]JobRun, 0, 32)
	
	for rows.Next() {
		var result JobRunDBRow
		var durationMicro sql.NullInt64

		err = rows.Scan(
			&result.JobID, &result.JobName, &result.FreqType, &result.Schedule, &result.NextRunTime, &result.JobStatus,
			&result.ScheduleType, &result.CreatedAt, &result.UpdatedAt,
			&result.ResultId, &result.StartTime, &durationMicro,
			&result.ResultStatus, &result.ErrorMsg, &result.RunNumber,
		)
		if err != nil {
			return nil, nil, serr.Wrap(err, "failed to scan result row")
		}

		jr := JobRun{
			JobID:        result.JobID,
			JobName:      result.JobName.String,
			FreqType:     result.FreqType.String,
			Schedule:     result.Schedule.String,
			NextRunTime:  result.NextRunTime.Time,
			JobStatus:    result.JobStatus.String,
			ScheduleType: result.ScheduleType.String,
			CreatedAt:    result.CreatedAt.Time,
			UpdatedAt:    result.UpdatedAt.Time,
			ResultId:     result.ResultId.Int64,
			StartTime:    result.StartTime.Time,
			ResultStatus: result.ResultStatus.String,
			ErrorMsg:     result.ErrorMsg.String,
			RunNumber:    int(result.RunNumber.Int64),
		}

		if durationMicro.Valid {
			jr.Duration = time.Duration(durationMicro.Int64) * time.Microsecond
		}

		results = append(results, jr)
	}


	return results, resultCounts, nil
}

// GetJobResultsPaginated retrieves paginated results for a specific job
func (s *DuckDBStore) GetJobResultsPaginated(jobID string, offset, limit int) ([]JobResult, int, error) {
	// Get total count
	var totalCount int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM job_results WHERE job_id = ?`, jobID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get paginated results
	rows, err := s.db.Query(`
		SELECT job_id, start_time, end_time, duration_micro, status, success_msg, error_msg
		FROM job_results 
		WHERE job_id = ?
		ORDER BY start_time DESC
		LIMIT ? OFFSET ?
	`, jobID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get job results: %w", err)
	}
	defer rows.Close()

	results := make([]JobResult, 0, limit)
	for rows.Next() {
		var result JobResult
		var durationMicro int64
		err := rows.Scan(
			&result.JobID, &result.StartTime, &result.EndTime, &durationMicro,
			&result.Status, &result.SuccessMsg, &result.ErrorMsg,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan result row: %w", err)
		}
		result.Duration = time.Duration(durationMicro) * time.Microsecond
		results = append(results, result)
	}

	return results, totalCount, nil
}

// GetJobRuns retrieves historical results for a job
// Set limit to 0 for all
func (s *DuckDBStore) GetJobRuns(limit int) ([]JobRun, error) {
	rows, err := s.db.Query(`
drop table if exists runs;
create temp table runs as (with results as (
       select result_id, job_id, start_time, duration_micro, status, error_msg
       from job_results
       )
       select j.job_id, null job_name, null frequency, null schedule, null next_run_time, null status,
              null schedule_type, j.created_at, null updated_at,
               r.result_id, r.start_time, r.duration_micro, r.status result_status, r.error_msg
       from results r join jobs j on r.job_id = j.job_id
       union all
       select j.job_id, j.job_name, case when j.schedule is null or j.schedule = '' then 'one-time' else j.schedule end as frequency, 
              j.schedule, j.next_run_time, j.status,
              j.schedule_type, j.created_at, j.updated_at,
               null as result_id, null as start_time, null as duration_micro, null as result_status, null as error_msg
      from jobs j);
select * from runs order by created_at desc, result_id desc nulls first
  LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get job results: %w", err)
	}
	defer rows.Close()

	results := make([]JobRun, 0, 32)

	// count := 0

	for rows.Next() {
		var result JobRunDBRow

		var durationMicro sql.NullInt64 // duration gets special handling

		err = rows.Scan(
			&result.JobID, &result.JobName, &result.FreqType, &result.Schedule, &result.NextRunTime, &result.JobStatus,
			&result.ScheduleType, &result.CreatedAt, &result.UpdatedAt,
			&result.ResultId, &result.StartTime, &durationMicro,
			&result.ResultStatus, &result.ErrorMsg,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan result row")
		}

		jr := JobRun{
			JobID:        result.JobID,
			JobName:      result.JobName.String,
			FreqType:     result.FreqType.String,
			Schedule:     result.Schedule.String,
			NextRunTime:  result.NextRunTime.Time,
			JobStatus:    result.JobStatus.String,
			ScheduleType: result.ScheduleType.String,
			CreatedAt:    result.CreatedAt.Time,
			UpdatedAt:    result.UpdatedAt.Time,
			ResultId:     result.ResultId.Int64,
			StartTime:    result.StartTime.Time,
			Duration:     time.Duration(durationMicro.Int64) * time.Microsecond,
			ResultStatus: result.ResultStatus.String,
			ErrorMsg:     result.ErrorMsg.String,
		}

		// if count < 15 {
		// 	fmt.Printf("**-> result %d -> %#v\n", count, result)
		// }
		results = append(results, jr)
	}

	return results, nil
}

// CleanupJobResults deletes job results older than the specified duration
func (s *DuckDBStore) CleanupJobResults(olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)

	result, err := s.db.Exec(`
		DELETE FROM job_results 
		WHERE end_time < ?
	`, cutoffTime)

	if err != nil {
		return fmt.Errorf("failed to cleanup old job results: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		fmt.Printf("Cleaned up %d job results older than %s\n", rowsAffected, olderThan)
	}

	return nil
}

// Close closes the database connection
func (s *DuckDBStore) Close() error {
	return s.db.Close()
}
