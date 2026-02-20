package kvdb

import (
	"os"
)

// KVDB represents a key-value database
type KVDB struct {
	path string
	file *os.File
	// TODO: Add necessary fields for your implementation
}

// Open initializes the database and associates it with the given path.
// If the path doesn't exist, creates an empty database.
// Returns error if operation fails.
func Open(path string) (*KVDB, error) {
	// TODO: Implement database opening logic
	// - Create file if not exists
	// - Open file with appropriate flags
	// - Initialize database structure
	// - Handle crash recovery if needed
	return nil, nil
}

// Put sets the value for the given key.
// If key already exists, overwrites the previous value.
// Returns error if operation fails.
func (db *KVDB) Put(key, value string) error {
	// TODO: Implement put operation
	// - Write key-value pair to database
	// - Ensure crash consistency (use fsync/fdatasync)
	// - Handle concurrent access (file locking)
	return nil
}

// Get retrieves the value for the given key.
// Returns the value and nil error if found.
// Returns empty string and error if key doesn't exist.
func (db *KVDB) Get(key string) (string, error) {
	// TODO: Implement get operation
	// - Search for key in database
	// - Return corresponding value
	// - Handle concurrent access
	return "", nil
}

// Close closes the database and releases associated resources.
// Returns error if operation fails.
func (db *KVDB) Close() error {
	// TODO: Implement database closing logic
	// - Flush any pending writes
	// - Close file descriptor
	// - Release locks
	return nil
}
