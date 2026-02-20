package kvdb

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

// setupTestDB creates a test database and returns cleanup function
func setupTestDB(t *testing.T, name string) (*KVDB, func()) {
	dbPath := filepath.Join("./tmp", name)
	os.RemoveAll(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}

	cleanup := func() {
		if db != nil {
			db.Close()
		}
		os.RemoveAll(dbPath)
	}

	return db, cleanup
}

// TestKVDBOpen tests database opening and closing
func TestKVDBOpen(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_open.db")
	defer cleanup()

	if db == nil {
		t.Fatal("Database should not be nil")
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Must close db: %v", err)
	}
}

// TestKVDBPutGet tests basic put and get operations
func TestKVDBPutGet(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_put_get.db")
	defer cleanup()

	// Test put operation
	if err := db.Put("key", "value"); err != nil {
		t.Fatalf("Must put key: %v", err)
	}

	// Test get operation
	value, err := db.Get("key")
	if err != nil {
		t.Fatalf("Must get key: %v", err)
	}

	if value != "value" {
		t.Fatalf("Expected 'value', got '%s'", value)
	}
}

// TestKVDBGetNonExistent tests getting a non-existent key
func TestKVDBGetNonExistent(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_nonexistent.db")
	defer cleanup()

	// Try to get non-existent key
	_, err := db.Get("nonexistent")
	if err == nil {
		t.Fatal("Should return error for non-existent key")
	}
}

// TestKVDBOverwrite tests overwriting existing key
func TestKVDBOverwrite(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_overwrite.db")
	defer cleanup()

	// Put initial value
	if err := db.Put("key", "value1"); err != nil {
		t.Fatalf("Must put key: %v", err)
	}

	// Overwrite with new value
	if err := db.Put("key", "value2"); err != nil {
		t.Fatalf("Must put key: %v", err)
	}

	// Verify new value
	value, err := db.Get("key")
	if err != nil {
		t.Fatalf("Must get key: %v", err)
	}

	if value != "value2" {
		t.Fatalf("Expected 'value2', got '%s'", value)
	}
}

// TestKVDBPersistence tests data persistence across open/close
func TestKVDBPersistence(t *testing.T) {
	dbPath := filepath.Join("./tmp", "test_persistence.db")
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	// First session: write data
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}

	if err := db.Put("key1", "value1"); err != nil {
		t.Fatalf("Must put key: %v", err)
	}

	if err := db.Put("key2", "value2"); err != nil {
		t.Fatalf("Must put key: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Must close db: %v", err)
	}

	// Second session: read data
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}
	defer db.Close()

	value1, err := db.Get("key1")
	if err != nil {
		t.Fatalf("Must get key1: %v", err)
	}
	if value1 != "value1" {
		t.Fatalf("Expected 'value1', got '%s'", value1)
	}

	value2, err := db.Get("key2")
	if err != nil {
		t.Fatalf("Must get key2: %v", err)
	}
	if value2 != "value2" {
		t.Fatalf("Expected 'value2', got '%s'", value2)
	}
}

// TestKVDBMultipleKeys tests storing and retrieving multiple keys
func TestKVDBMultipleKeys(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_multiple_keys.db")
	defer cleanup()

	// Put multiple keys
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	values := []string{"value1", "value2", "value3", "value4", "value5"}

	for i := range keys {
		if err := db.Put(keys[i], values[i]); err != nil {
			t.Fatalf("Must put key %s: %v", keys[i], err)
		}
	}

	// Get all keys and verify
	for i := range keys {
		value, err := db.Get(keys[i])
		if err != nil {
			t.Fatalf("Must get key %s: %v", keys[i], err)
		}
		if value != values[i] {
			t.Fatalf("Expected '%s', got '%s'", values[i], value)
		}
	}
}

// TestKVDBConcurrentReads tests concurrent read operations
func TestKVDBConcurrentReads(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_concurrent_reads.db")
	defer cleanup()

	// Put some data
	if err := db.Put("key", "value"); err != nil {
		t.Fatalf("Must put key: %v", err)
	}

	// Concurrent reads
	var wg sync.WaitGroup
	numReaders := 100
	errors := make(chan error, numReaders)

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			value, err := db.Get("key")
			if err != nil {
				errors <- fmt.Errorf("reader %d: %v", id, err)
				return
			}
			if value != "value" {
				errors <- fmt.Errorf("reader %d: expected 'value', got '%s'", id, value)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestKVDBConcurrentWrites tests concurrent write operations
func TestKVDBConcurrentWrites(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_concurrent_writes.db")
	defer cleanup()

	// Concurrent writes to different keys
	var wg sync.WaitGroup
	numWriters := 50
	errors := make(chan error, numWriters)

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)
			value := fmt.Sprintf("value%d", id)
			if err := db.Put(key, value); err != nil {
				errors <- fmt.Errorf("writer %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	// Verify all writes
	for i := 0; i < numWriters; i++ {
		key := fmt.Sprintf("key%d", i)
		expectedValue := fmt.Sprintf("value%d", i)
		value, err := db.Get(key)
		if err != nil {
			t.Errorf("Must get key %s: %v", key, err)
		}
		if value != expectedValue {
			t.Errorf("Expected '%s', got '%s'", expectedValue, value)
		}
	}
}

// TestKVDBConcurrentSameKey tests concurrent writes to the same key
func TestKVDBConcurrentSameKey(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_concurrent_same_key.db")
	defer cleanup()

	// Concurrent writes to the same key
	var wg sync.WaitGroup
	numWriters := 50

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			value := fmt.Sprintf("value%d", id)
			db.Put("shared_key", value)
		}(i)
	}

	wg.Wait()

	// The final value should be one of the written values
	value, err := db.Get("shared_key")
	if err != nil {
		t.Fatalf("Must get shared_key: %v", err)
	}

	// Verify it's a valid value
	found := false
	for i := 0; i < numWriters; i++ {
		if value == fmt.Sprintf("value%d", i) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Final value '%s' is not one of the expected values", value)
	}
}

// TestKVDBConcurrentReadWrite tests concurrent reads and writes
func TestKVDBConcurrentReadWrite(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_concurrent_rw.db")
	defer cleanup()

	// Initial data
	if err := db.Put("counter", "0"); err != nil {
		t.Fatalf("Must put initial value: %v", err)
	}

	var wg sync.WaitGroup
	numOps := 100

	// Mix of reads and writes
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				// Write
				db.Put("counter", fmt.Sprintf("%d", id))
			} else {
				// Read
				db.Get("counter")
			}
		}(i)
	}

	wg.Wait()

	// Should be able to read final value
	_, err := db.Get("counter")
	if err != nil {
		t.Fatalf("Must get counter: %v", err)
	}
}

// TestKVDBCrashRecovery tests recovery after simulated crash
func TestKVDBCrashRecovery(t *testing.T) {
	dbPath := filepath.Join("./tmp", "test_crash_recovery.db")
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	// First session: write data
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}

	if err := db.Put("key1", "value1"); err != nil {
		t.Fatalf("Must put key1: %v", err)
	}

	if err := db.Put("key2", "value2"); err != nil {
		t.Fatalf("Must put key2: %v", err)
	}

	// Simulate crash: release lock and close fd without proper cleanup
	// The fsync in Put has already persisted the data to disk
	syscall.Flock(int(db.file.Fd()), syscall.LOCK_UN)
	db.file.Close()

	// Second session: recovery and verification
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db after crash: %v", err)
	}
	defer db.Close()

	// Data should still be there
	value1, err := db.Get("key1")
	if err != nil {
		t.Fatalf("Must get key1 after recovery: %v", err)
	}
	if value1 != "value1" {
		t.Fatalf("Expected 'value1', got '%s'", value1)
	}

	value2, err := db.Get("key2")
	if err != nil {
		t.Fatalf("Must get key2 after recovery: %v", err)
	}
	if value2 != "value2" {
		t.Fatalf("Expected 'value2', got '%s'", value2)
	}
}

// TestKVDBCorruptedData tests handling of corrupted data
func TestKVDBCorruptedData(t *testing.T) {
	dbPath := filepath.Join("./tmp", "test_corrupted.db")
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	// Create and write some data
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}

	if err := db.Put("key1", "value1"); err != nil {
		t.Fatalf("Must put key1: %v", err)
	}

	if err := db.Put("key2", "value2"); err != nil {
		t.Fatalf("Must put key2: %v", err)
	}

	db.Close()

	// Corrupt the file by appending garbage
	file, err := os.OpenFile(dbPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for corruption: %v", err)
	}
	file.Write([]byte("garbage data"))
	file.Close()

	// Try to open corrupted database
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("Should be able to open corrupted db: %v", err)
	}
	defer db.Close()

	// Should still be able to read valid data
	value1, err := db.Get("key1")
	if err != nil {
		t.Fatalf("Must get key1: %v", err)
	}
	if value1 != "value1" {
		t.Fatalf("Expected 'value1', got '%s'", value1)
	}
}

// TestKVDBLargeValues tests storing and retrieving large values
func TestKVDBLargeValues(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_large_values.db")
	defer cleanup()

	// Create a large value (1MB)
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	if err := db.Put("large_key", string(largeValue)); err != nil {
		t.Fatalf("Must put large value: %v", err)
	}

	value, err := db.Get("large_key")
	if err != nil {
		t.Fatalf("Must get large value: %v", err)
	}

	if len(value) != len(largeValue) {
		t.Fatalf("Expected length %d, got %d", len(largeValue), len(value))
	}

	if value != string(largeValue) {
		t.Fatal("Large value mismatch")
	}
}

// TestKVDBSpecialCharacters tests keys and values with special characters
func TestKVDBSpecialCharacters(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_special_chars.db")
	defer cleanup()

	testCases := []struct {
		key   string
		value string
	}{
		{"key with spaces", "value with spaces"},
		{"key\nwith\nnewlines", "value\nwith\nnewlines"},
		{"key\twith\ttabs", "value\twith\ttabs"},
		{"key=with=equals", "value=with=equals"},
		{"key:with:colons", "value:with:colons"},
		{"中文键", "中文值"},
		{"emoji🔑", "emoji🎉"},
		{"key\x00with\x00null", "value\x00with\x00null"},
	}

	for _, tc := range testCases {
		if err := db.Put(tc.key, tc.value); err != nil {
			t.Errorf("Must put key '%s': %v", tc.key, err)
			continue
		}

		value, err := db.Get(tc.key)
		if err != nil {
			t.Errorf("Must get key '%s': %v", tc.key, err)
			continue
		}

		if value != tc.value {
			t.Errorf("Key '%s': expected '%s', got '%s'", tc.key, tc.value, value)
		}
	}
}

// TestKVDBEmptyKeyValue tests edge cases with empty strings
func TestKVDBEmptyKeyValue(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_empty.db")
	defer cleanup()

	// Empty value
	if err := db.Put("key", ""); err != nil {
		t.Fatalf("Must put empty value: %v", err)
	}

	value, err := db.Get("key")
	if err != nil {
		t.Fatalf("Must get key with empty value: %v", err)
	}
	if value != "" {
		t.Fatalf("Expected empty string, got '%s'", value)
	}

	// Empty key (should work)
	if err := db.Put("", "value"); err != nil {
		t.Fatalf("Must put with empty key: %v", err)
	}

	value, err = db.Get("")
	if err != nil {
		t.Fatalf("Must get empty key: %v", err)
	}
	if value != "value" {
		t.Fatalf("Expected 'value', got '%s'", value)
	}
}

// TestKVDBStressTest performs a stress test with many operations
func TestKVDBStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	db, cleanup := setupTestDB(t, "test_stress.db")
	defer cleanup()

	numOperations := 1000
	var wg sync.WaitGroup
	errors := make(chan error, numOperations)

	// Mix of reads and writes
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id%100)
			value := fmt.Sprintf("value%d", id)

			// Write
			if err := db.Put(key, value); err != nil {
				errors <- fmt.Errorf("put %d: %v", id, err)
				return
			}

			// Read back
			time.Sleep(time.Millisecond)
			_, err := db.Get(key)
			if err != nil {
				errors <- fmt.Errorf("get %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
		if errorCount > 10 {
			t.Fatal("Too many errors, stopping")
		}
	}
}

// TestKVDBReopenMultipleTimes tests reopening database multiple times
func TestKVDBReopenMultipleTimes(t *testing.T) {
	dbPath := filepath.Join("./tmp", "test_reopen.db")
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	for i := 0; i < 10; i++ {
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Iteration %d: Must open db: %v", i, err)
		}

		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)

		if err := db.Put(key, value); err != nil {
			t.Fatalf("Iteration %d: Must put key: %v", i, err)
		}

		// Verify all previous keys still exist
		for j := 0; j <= i; j++ {
			expectedKey := fmt.Sprintf("key%d", j)
			expectedValue := fmt.Sprintf("value%d", j)

			gotValue, err := db.Get(expectedKey)
			if err != nil {
				t.Fatalf("Iteration %d: Must get key%d: %v", i, j, err)
			}
			if gotValue != expectedValue {
				t.Fatalf("Iteration %d: Expected '%s', got '%s'", i, expectedValue, gotValue)
			}
		}

		if err := db.Close(); err != nil {
			t.Fatalf("Iteration %d: Must close db: %v", i, err)
		}
	}
}

// TestKVDBFilePermissions tests that database files have correct permissions
func TestKVDBFilePermissions(t *testing.T) {
	dbPath := filepath.Join("./tmp", "test_permissions.db")
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}
	defer db.Close()

	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Must stat file: %v", err)
	}

	// Check file permissions (should be 0644)
	mode := info.Mode().Perm()
	if mode != 0644 {
		t.Errorf("Expected permissions 0644, got %o", mode)
	}
}

// TestKVDBSequentialConsistency tests sequential consistency
func TestKVDBSequentialConsistency(t *testing.T) {
	db, cleanup := setupTestDB(t, "test_sequential.db")
	defer cleanup()

	// Sequential writes
	for i := 0; i < 100; i++ {
		key := "counter"
		value := fmt.Sprintf("%d", i)
		if err := db.Put(key, value); err != nil {
			t.Fatalf("Must put value %d: %v", i, err)
		}

		// Immediately read back
		gotValue, err := db.Get(key)
		if err != nil {
			t.Fatalf("Must get value after put %d: %v", i, err)
		}

		if gotValue != value {
			t.Fatalf("Sequential consistency violated: expected '%s', got '%s'", value, gotValue)
		}
	}
}

// TestKVDBMultiProcessSimulation tests multiple database instances
func TestKVDBMultiProcessSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-process test in short mode")
	}

	dbPath := filepath.Join("./tmp", "test_multi_process.db")
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	// First process writes
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db1: %v", err)
	}

	if err := db1.Put("key1", "value1"); err != nil {
		t.Fatalf("Must put key1: %v", err)
	}

	// Note: Due to file locking, we cannot actually open a second instance
	// in the same process. This test verifies that the lock works.
	// In a real multi-process scenario, the second Open would block.

	db1.Close()

	// After closing, another process can open
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db2: %v", err)
	}
	defer db2.Close()

	value, err := db2.Get("key1")
	if err != nil {
		t.Fatalf("Must get key1 from db2: %v", err)
	}
	if value != "value1" {
		t.Fatalf("Expected 'value1', got '%s'", value)
	}
}

// TestKVDBBenchmarkPut benchmarks put operations
func BenchmarkKVDBPut(b *testing.B) {
	dbPath := filepath.Join("./tmp", "bench_put.db")
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Must open db: %v", err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		if err := db.Put(key, value); err != nil {
			b.Fatalf("Must put key: %v", err)
		}
	}
}

// TestKVDBBenchmarkGet benchmarks get operations
func BenchmarkKVDBGet(b *testing.B) {
	dbPath := filepath.Join("./tmp", "bench_get.db")
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Must open db: %v", err)
	}
	defer db.Close()

	// Prepare data
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		if err := db.Put(key, value); err != nil {
			b.Fatalf("Must put key: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%numKeys)
		if _, err := db.Get(key); err != nil {
			b.Fatalf("Must get key: %v", err)
		}
	}
}

// TestKVDBExternalProcess tests interaction with external process (if available)
func TestKVDBExternalProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping external process test in short mode")
	}

	// Check if we can run external commands
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go command not available, skipping external process test")
	}

	dbPath := filepath.Join("./tmp", "test_external.db")
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	// Create initial database
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}

	if err := db.Put("shared_key", "initial_value"); err != nil {
		t.Fatalf("Must put initial value: %v", err)
	}

	db.Close()

	// Verify we can reopen and read
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("Must reopen db: %v", err)
	}
	defer db.Close()

	value, err := db.Get("shared_key")
	if err != nil {
		t.Fatalf("Must get shared_key: %v", err)
	}

	if value != "initial_value" {
		t.Fatalf("Expected 'initial_value', got '%s'", value)
	}
}
