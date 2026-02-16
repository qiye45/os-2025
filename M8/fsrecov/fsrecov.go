package main

import (
	"flag"
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
	fsImage := flag.String("image", "fsrecov.img", "FAT32 file system image path")
	flag.Parse()

	os.Stdout.Sync()

	data, header, err := mapDisk(*fsImage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", *fsImage, err)
		os.Exit(1)
	}
	fmt.Printf("header %+v , data size:%d\n", header, len(data))

	// TODO: fsrecov implementation
	//	1.扫描所有簇
}

// 扫描所有簇
func scanClusters(data []byte, header *FAT32Header) {

}

// 扫描每一页，是否是目录文件或图片文件

// 读取目录文件

// 读取图片文件
