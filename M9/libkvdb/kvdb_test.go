package kvdb

import (
	"os"
	"testing"
)

// TestKVDBOpen tests database opening and closing
func TestKVDBOpen(t *testing.T) {
	dbPath := "./tmp/test.db"

	// Clean up before test
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Must close db: %v", err)
	}
}

// TestKVDBPutGet tests basic put and get operations
func TestKVDBPutGet(t *testing.T) {
	dbPath := "./tmp/test.db"

	// Clean up before test
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}
	defer db.Close()

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
	dbPath := "./tmp/test.db"

	// Clean up before test
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}
	defer db.Close()

	// Try to get non-existent key
	_, err = db.Get("nonexistent")
	if err == nil {
		t.Fatal("Should return error for non-existent key")
	}
}

// TestKVDBOverwrite tests overwriting existing key
func TestKVDBOverwrite(t *testing.T) {
	dbPath := "./tmp/test.db"

	// Clean up before test
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Must open db: %v", err)
	}
	defer db.Close()

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
	dbPath := "./tmp/test.db"

	// Clean up before test
	os.Remove(dbPath)
	defer os.Remove(dbPath)

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
