package jobpro

import (
	"log"
	"os"

	"github.com/rohanthewiz/logger"
)

// Init initializes the job processor with a DuckDB store and a job manager
// TODO: Pass duckdb path as an argument instead of using env var.
//
//	If dbPath is empty, use as in-memory database
func Init() (manager *DefaultJobManager, store *DuckDBStore) {
	log.Println("Starting job processor")

	// Initialize DuckDB store
	dbPath := os.Getenv("DUCKDB_PATH")
	if dbPath == "" {
		dbPath = "jobs.duckdb"
	}

	store, err := NewDuckDBStore(dbPath)
	if err != nil {
		logger.LogErr(err, "Failed to initialize DuckDB store")
		os.Exit(1)
	}

	// Initialize job manager
	return NewJobManager(store), store
}
