package main

// refer https://jyywiki.cn/OS/manuals/MSFAT-spec.pdf

type FAT32Header struct {
	BsJmpBoot [3]byte
	BsOemName [8]byte
	// uint16对齐系数是2，11 %2 !=0，会填充1个字节
	BpbBytesPerSec uint16    // 每扇区字节数（通常 512）
	BpbSecPerClus  uint8     // 每簇扇区数
	BpbRsvdSecCnt  uint16    // 保留扇区数（引导扇区所在区域）
	BpbNumFats     uint8     // FAT 表数量（通常为 2）
	BpbRootEntCnt  uint16    // 根目录条目数（FAT32 中为 0）
	BpbTotSec16    uint16    // 总扇区数（16位，FAT32 中为 0）
	BpbMedia       uint8     // 媒体描述符
	BpbFatSz16     uint16    // FAT 表大小（16位，FAT32 中为 0）
	BpbSecPerTrk   uint16    // 每磁道扇区数
	BpbNumHeads    uint16    // 磁头数
	BpbHiddSec     uint32    // 隐藏扇区数
	BpbTotSec32    uint32    // 总扇区数（32位）
	BpbFatSz32     uint32    // FAT 表大小（32位）
	BpbExtFlags    uint16    // 扩展标志
	BpbFsVer       uint16    // 文件系统版本
	BpbRootClus    uint32    // 根目录起始簇号（通常为 2）
	BpbFsInfo      uint16    // FSInfo 结构扇区号
	BpbBkBootSec   uint16    // 备份引导扇区位置
	BpbReserved    [12]byte  // 保留字段（12字节）
	BsDrvNum       uint8     // 驱动器号
	BsReserved1    uint8     // 保留（用于 Windows NT）
	BsBootSig      uint8     // 扩展引导签名（0x29）
	BsVolId        uint32    // 卷序列号
	BsVolLab       [11]byte  // 卷标（11字节）
	BsFilSysType   [8]byte   // 文件系统类型（"FAT32   "）
	Padding        [420]byte // 填充到 510 字节，改为导出字段
	SignatureWord  uint16    // 引导扇区签名（0xAA55）
}

// FAT32DirEntry 文件目录项 (32 bytes)
// refer Section 6: Directory Structure / Section 7: Long File Name Implementation (optional)
type FAT32DirEntry struct {
	DirName         [11]byte // 文件名（8.3 格式，11字节）
	DirAttr         uint8    // 文件属性（只读、隐藏、系统等）
	DirNtRes        uint8    // Windows NT 保留字段
	DirCrtTimeTenth uint8    // 创建时间的十分之一秒
	DirCrtTime      uint16   // 创建时间
	DirCrtDate      uint16   // 创建日期
	DirLastAccDate  uint16   // 最后访问日期
	DirFstClusHi    uint16   // 起始簇号高 16 位
	DirWrtTime      uint16   // 最后修改时间
	DirWrtDate      uint16   // 最后修改日期
	DirFstClusLo    uint16   // 起始簇号低 16 位
	DirFileSize     uint32   // 文件大小（字节）
}

// FAT32LFNEntry 长文件名目录项 (32 bytes)
type FAT32LFNEntry struct {
	Ord       uint8    // 序列号
	Name1     [10]byte // 5 chars (UTF-16LE)
	Attr      uint8
	Type      uint8
	Checksum  uint8
	Name2     [12]byte // 6 chars
	FstClusLO uint16   // Must be 0
	Name3     [4]byte  // 2 chars
}

// BMPHeader BMP 文件头
type BMPHeader struct {
	Signature    uint16 // "BM"
	FileSize     uint32
	Reserved     uint32
	DataOffset   uint32
	HeaderSize   uint32
	Width        int32
	Height       int32
	Planes       uint16
	BitsPerPixel uint16
}

// 目录项属性
const (
	AttrReadOnly  = 0x01
	AttrHidden    = 0x02
	AttrSystem    = 0x04
	AttrVolumeID  = 0x08
	AttrDirectory = 0x10
	AttrArchive   = 0x20
	AttrLongName  = AttrReadOnly | AttrHidden | AttrSystem | AttrVolumeID
)
