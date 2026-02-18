package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	ClusterUnused = iota
	ClusterDirectory
	ClusterBMPHeader
	ClusterBMPData
)

const (
	DirEntrySize = 32
)

type ClusterInfo struct {
	ClusterNum int
	Type       int
	Data       []byte
}

type FileInfo struct {
	Name         string
	StartCluster int64
	Size         int64
}

func mapDisk(filename string) ([]byte, *FAT32Header, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %v", err)
	}

	if len(data) < 512 {
		return nil, nil, fmt.Errorf("file too small for FAT32 header")
	}

	// 因为go内存对齐，会导致数据对齐错误
	// header := (*FAT32Header)(unsafe.Pointer(&data[0]))
	header := &FAT32Header{}
	buf := bytes.NewReader(data[:512])

	// 使用 binary.Read 按小端序读取整个结构体
	if err := binary.Read(buf, binary.LittleEndian, header); err != nil {
		return nil, nil, fmt.Errorf("failed to parse header: %v", err)
	}

	if header.SignatureWord != 0xaa55 {
		return nil, nil, fmt.Errorf("not a valid FAT file image: signature = 0x%x", header.SignatureWord)
	}

	expectedSize := int64(header.BpbTotSec32) * int64(header.BpbBytesPerSec)
	if int64(len(data)) != expectedSize {
		return nil, nil, fmt.Errorf("file size mismatch: expected %d, got %d", expectedSize, len(data))
	}

	return data, header, nil
}

func getClusterOffset(header *FAT32Header, clusterNum int64) int64 {
	// 数据区开始位置 = 保留扇区 + FAT表数量 * FAT表大小
	dataStart := int64(header.BpbRsvdSecCnt)*int64(header.BpbBytesPerSec) + int64(header.BpbNumFats)*int64(header.BpbFatSz32)*int64(header.BpbBytesPerSec)
	// 簇号的偏移 = 数据区开始位置 + (簇号-2) * 每簇字节数
	clusterOffset := dataStart + (clusterNum-2)*int64(header.BpbSecPerClus)*int64(header.BpbBytesPerSec)
	return clusterOffset
}
func getClusterSize(header *FAT32Header) int64 {
	return int64(header.BpbBytesPerSec) * int64(header.BpbSecPerClus)
}

// 扫描所有簇
func scanClusters(data []byte, header *FAT32Header) []ClusterInfo {
	clusterSize := getClusterSize(header)
	totalClusters := int64(header.BpbTotSec32) / int64(header.BpbSecPerClus)
	clusters := make([]ClusterInfo, 0, totalClusters)
	for i := int64(2); i < totalClusters; i++ {
		offset := getClusterOffset(header, i)
		if offset+clusterSize > int64(len(data)) {
			break
		}
		clusterData := data[offset : offset+clusterSize]
		clusterType := classifyCluster(clusterData)
		if clusterType != ClusterUnused {
			clusters = append(clusters, ClusterInfo{
				ClusterNum: int(i),
				Type:       clusterType,
				Data:       clusterData,
			})
		}
	}
	return clusters
}

func classifyCluster(data []byte) int {
	// 是否BMP头
	if len(data) >= 2 && data[0] == BMPSignature1 && data[1] == BMPSignature2 {
		return ClusterBMPHeader
	}
	// 是否目录
	pattern := []byte("BMP")
	if bytes.Count(data, pattern) > 1 {
		return ClusterDirectory
	}
	return ClusterUnused
}

func parseDirectoryEntries(clusters []ClusterInfo) []FileInfo {
	files := make([]FileInfo, 0)

	for _, cluster := range clusters {
		if cluster.Type != ClusterDirectory {
			continue
		}
		//	每个目录项32字节
		numEntries := len(cluster.Data) / DirEntrySize
		lfnParts := make([]string, 0)
		for i := 0; i < numEntries; i++ {
			offset := i * DirEntrySize
			entryData := cluster.Data[offset : offset+DirEntrySize]
			attr := entryData[11]
			if attr == AttrLongName {
				// 解析长文件名条目
				lfn := parseLongFileEntry(entryData)
				if lfn == nil {
					continue
				}
				// lfn是逆序存放
				lfnParts = append([]string{decodeUTF16LE(append(lfn.Name1[:], append(lfn.Name2[:], lfn.Name3[:]...)...))}, lfnParts...)
			} else {
				// 这是短文件名条目，可能是：
				// 1. 子目录
				// 2. 普通文件 ✓
				// 3. 卷标
				if (attr&AttrDirectory) != 0 || (attr&AttrVolumeID) != 0 {
					lfnParts = nil
					continue // 跳过！我不关心文件夹，我只关心文件
				}
				sfn := parseShortFileEntry(entryData)
				if sfn == nil {
					continue
				}
				var filename string
				if len(lfnParts) > 0 {
					// 使用长文件名就行，短文件名为了兼容
					filename = strings.Join(lfnParts, "")
					lfnParts = nil
				} else {
					name := string(bytes.TrimSpace(sfn.DirName[0:8]))
					ext := string(bytes.TrimSpace(sfn.DirName[8:11]))
					filename = name + "." + ext
				}
				startCluster := (int64(sfn.DirFstClusHi) << 16) | int64(sfn.DirFstClusLo)
				size := sfn.DirFileSize
				if startCluster > 0 && sfn.DirFileSize > 0 && strings.Contains(filename, ".bmp") {
					files = append(files, FileInfo{
						Name:         filename,
						StartCluster: startCluster,
						Size:         int64(size),
					})
				}

			}
		}
	}

	return files
}

func parseLongFileEntry(data []byte) *FAT32LFNEntry {
	entry := &FAT32LFNEntry{}
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, entry)
	if err != nil {
		return nil
	}
	return entry
}

func parseShortFileEntry(data []byte) *FAT32DirEntry {
	entry := &FAT32DirEntry{}
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, entry)
	if err != nil {
		return nil
	}
	return entry
}

// decodeUTF16LE 将 UTF-16LE 字节序列转换为字符串
func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		return ""
	}
	var result []rune
	for i := 0; i < len(data); i += 2 {
		char := uint16(data[i]) | (uint16(data[i+1]) << 8)
		if char == 0 || char == 0xFFFF {
			break // 遇到终止符
		}
		result = append(result, rune(char))
	}
	return string(result)
}

// 恢复BMP文件
func recoverBMPFile(fileInfo FileInfo, clusters []ClusterInfo, header *FAT32Header, data []byte) ([]byte, error) {
	fileData := make([]byte, 0, fileInfo.Size)
	offset := getClusterOffset(header, int64(fileInfo.StartCluster))
	// 策略1：假设文件连续存储 文件准确率 47.42% 大小准确率 98.97%
	fileData = append(fileData, data[offset:min(offset+fileInfo.Size, int64(len(data)))]...)
	return fileData, nil
}

func calculateSHA1(data []byte) string {
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:])
}

func main() {
	fsImage := flag.String("image", "fsrecov.img", "FAT32 file system image path")
	flag.Parse()

	data, header, err := mapDisk(*fsImage)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", *fsImage, err)
		os.Exit(1)
	}
	//fmt.Printf("header %+v , data size:%d\n", header, len(data))

	// 1.扫描所有簇
	clusters := scanClusters(data, header)
	// 2.解析目录项
	files := parseDirectoryEntries(clusters)
	// 3.恢复BMP文件
	for _, file := range files {
		bmpData, err := recoverBMPFile(file, clusters, header, data)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", file.Name, err)
			continue
		}

		sha1sum := calculateSHA1(bmpData)
		fmt.Printf("%s %s %d\n", file.Name, sha1sum, file.Size)
	}
}
