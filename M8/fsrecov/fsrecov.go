package main

import (
	"fmt"
	"io"
	"os"
	"unsafe"
)

func mapDisk(fname string) ([]byte, *FAT32Header, error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file info: %v", err)
	}
	size := fileInfo.Size()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %v", err)
	}

	if len(data) < int(unsafe.Sizeof(FAT32Header{})) {
		return nil, nil, fmt.Errorf("file too small for FAT32 header")
	}

	header := (*FAT32Header)(unsafe.Pointer(&data[0]))

	if header.Signature_word != 0xaa55 {
		return nil, nil, fmt.Errorf("not a valid FAT file image: signature = 0x%x", header.Signature_word)
	}

	expectedSize := int64(header.BPB_TotSec32) * int64(header.BPB_BytsPerSec)
	if size != expectedSize {
		return nil, nil, fmt.Errorf("file size mismatch: expected %d, got %d", expectedSize, size)
	}

	return data, header, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s fs-image\n", os.Args[0])
		os.Exit(1)
	}

	os.Stdout.Sync()

	data, header, err := mapDisk(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	// TODO: fsrecov implementation
	_ = data
	_ = header
}
