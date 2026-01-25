package main

import (
	"os"
	"testing"
	"unsafe"
)

// TestVersion tests the version of fsrecov
func TestVersion(t *testing.T) {
	// TODO: Add tests here
	// This test should verify that fsrecov can be run with a test image file
	// For now, we'll just check that the test compiles
}

// TestMapDisk tests the mapDisk function
func TestMapDisk(t *testing.T) {
	// Create a minimal FAT32 image for testing
	testFile := "test_fs.img"
	defer os.Remove(testFile)

	// Create a 1MB test file filled with zeros
	data := make([]byte, 1024*1024)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test that mapDisk properly rejects invalid images
	_, _, err := mapDisk(testFile)
	if err == nil {
		t.Error("mapDisk should reject non-FAT32 images")
	}
}

// TestFAT32HeaderSize tests that FAT32Header is exactly 512 bytes
func TestFAT32HeaderSize(t *testing.T) {
	if unsafe.Sizeof(FAT32Header{}) != 512 {
		t.Errorf("FAT32Header size is %d, expected 512", unsafe.Sizeof(FAT32Header{}))
	}
}

// TestFAT32DirEntrySize tests that FAT32DirEntry is exactly 32 bytes
func TestFAT32DirEntrySize(t *testing.T) {
	if unsafe.Sizeof(FAT32DirEntry{}) != 32 {
		t.Errorf("FAT32DirEntry size is %d, expected 32", unsafe.Sizeof(FAT32DirEntry{}))
	}
}
