package main

import (
	"unsafe"
)

type FAT32Header struct {
	BS_jmpBoot     [3]byte
	BS_OEMName     [8]byte
	BPB_BytsPerSec uint16
	BPB_SecPerClus uint8
	BPB_RsvdSecCnt uint16
	BPB_NumFATs    uint8
	BPB_RootEntCnt uint16
	BPB_TotSec16   uint16
	BPB_Media      uint8
	BPB_FATSz16    uint16
	BPB_SecPerTrk  uint16
	BPB_NumHeads   uint16
	BPB_HiddSec    uint32
	BPB_TotSec32   uint32
	BPB_FATSz32    uint32
	BPB_ExtFlags   uint16
	BPB_FSVer      uint16
	BPB_RootClus   uint32
	BPB_FSInfo     uint16
	BPB_BkBootSec  uint16
	BPB_Reserved   [12]byte
	BS_DrvNum      uint8
	BS_Reserved1   uint8
	BS_BootSig     uint8
	BS_VolID       uint32
	BS_VolLab      [11]byte
	BS_FilSysType  [8]byte
	__padding_1    [420]byte
	Signature_word uint16
}

type FAT32DirEntry struct {
	DIR_Name         [11]byte
	DIR_Attr         uint8
	DIR_NTRes        uint8
	DIR_CrtTimeTenth uint8
	DIR_CrtTime      uint16
	DIR_CrtDate      uint16
	DIR_LastAccDate  uint16
	DIR_FstClusHI    uint16
	DIR_WrtTime      uint16
	DIR_WrtDate      uint16
	DIR_FstClusLO    uint16
	DIR_FileSize     uint32
}

const (
	CLUS_INVALID = 0xffffff7

	ATTR_READ_ONLY = 0x01
	ATTR_HIDDEN    = 0x02
	ATTR_SYSTEM    = 0x04
	ATTR_VOLUME_ID = 0x08
	ATTR_DIRECTORY = 0x10
	ATTR_ARCHIVE   = 0x20
)

func init() {
	if unsafe.Sizeof(FAT32Header{}) != 512 {
		panic("FAT32Header size must be 512 bytes")
	}
}
