package mymalloc

import (
	"sync/atomic"
	"syscall"
	"unsafe"
)

const (
	locked       = 1
	unlocked     = 0
	pageSize     = 4096
	headerSize   = 8         // 头部存放block的大小。去掉哈希表，优化并发读取问题
	maxSmallSize = 32768     // 小对象最大阈值 32KB，超过走 mmap
	chunkSize    = 64 * 1024 // 每次向 OS 申请的块大小 (64KB)
	shardCount   = 64        // 分片数量
)

// SpinLock 自旋锁实现
type SpinLock struct {
	status int32
}

func (s *SpinLock) Lock() {
	for !atomic.CompareAndSwapInt32(&s.status, unlocked, locked) {
		// 自旋等待
	}
}

func (s *SpinLock) Unlock() {
	atomic.StoreInt32(&s.status, unlocked)
}

// 全局状态
var MallocCount int64             //用于测试的计数器
var shards [shardCount]MemoryPool // 全局分片堆
var sizeClasses []int             // 内存规格
var shardCtr uint64               // 轮询负载

func init() {
	for i := 16; i <= maxSmallSize; i <<= 1 {
		sizeClasses = append(sizeClasses, i)
	}

	for i := range shards {
		shards[i].init()
	}
}

type MemoryPool struct {
	lock     SpinLock
	page     unsafe.Pointer
	offset   int
	freeList [][]unsafe.Pointer
}

func (m *MemoryPool) init() {
	m.freeList = make([][]unsafe.Pointer, len(sizeClasses))
}

func (m *MemoryPool) malloc(size, classIdx int) unsafe.Pointer {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 从空闲链表查找，保证空闲块的大小是>=申请的size的块

	if len(m.freeList[classIdx]) > 0 {
		idx := len(m.freeList[classIdx]) - 1
		p := m.freeList[classIdx][idx]
		m.freeList[classIdx] = m.freeList[classIdx][:idx]
		return p
	}

	// 从当前 Chunk 切分 (Bump Allocator)
	// 如果当前 chunk 剩余空间不足，申请新的大块 (Region)
	if m.page == nil || m.offset+size > chunkSize {
		m.page = Vmalloc(nil, chunkSize)
		m.offset = 0
	}
	p := unsafe.Pointer(uintptr(m.page) + uintptr(m.offset))
	m.offset += size
	return p
}

func (m *MemoryPool) free(p unsafe.Pointer, classIdx int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	// 小块添加到空闲链表
	m.freeList[classIdx] = append(m.freeList[classIdx], p)
}

func getFixSize(size int) (int, int) {
	for i, class := range sizeClasses {
		if size <= class {
			return class, i
		}
	}
	return sizeClasses[len(sizeClasses)-1], len(sizeClasses) - 1
}

// Mymalloc 分配 size 字节的内存
// 返回 8 字节对齐的内存地址，失败返回 nil
func Mymalloc(size int) unsafe.Pointer {
	if size <= 0 {
		return nil
	}
	atomic.AddInt64(&MallocCount, 1)
	allocSize := size + headerSize

	// 大块直接mmap
	// 这样避免大对象阻塞分片锁，也避免小对象堆产生过多碎片
	if allocSize > maxSmallSize {
		pages := (allocSize + pageSize - 1) / pageSize
		realSize := pages * pageSize
		p := Vmalloc(nil, realSize)
		*(*int)(p) = realSize
		return unsafe.Pointer(uintptr(p) + uintptr(headerSize))
	}

	// 小对象走分片的内存池
	fixSize, classIdx := getFixSize(size)
	shardIdx := atomic.AddUint64(&shardCtr, 1) % shardCount
	p := shards[shardIdx].malloc(fixSize, classIdx)
	*(*int)(p) = fixSize
	return unsafe.Pointer(uintptr(p) + uintptr(headerSize))
}

// Myfree 释放之前分配的内存
func Myfree(p unsafe.Pointer) {
	headerPtr := unsafe.Pointer(uintptr(p) - uintptr(headerSize))
	size := *(*int)(headerPtr)
	// 大块直接释放
	if size > maxSmallSize {
		// 应该使用headerPtr的指针
		Vmfree(headerPtr, size)
		return
	}
	_, classIdx := getFixSize(size)

	// 随机选一个分片归还
	shardIdx := uintptr(headerPtr) % shardCount
	shards[shardIdx].free(headerPtr, classIdx)
}

// Reset 暴力重置 (仅用于测试环境清理)
func Reset() {
	// 需要锁住所有分片进行重置
	for i := 0; i < shardCount; i++ {
		s := &shards[i]
		s.lock.Lock()
		s.page = nil
		s.offset = chunkSize // 强制下次 malloc 分配新页
		for j := range s.freeList {
			s.freeList[j] = s.freeList[j][:0]
		}
		s.lock.Unlock()
	}
	atomic.StoreInt64(&MallocCount, 0)
}

// Vmalloc 申请大块内存（length 必须是 4096 的倍数）
func Vmalloc(addr unsafe.Pointer, length int) unsafe.Pointer {
	// 使用 mmap 系统调用
	prot := syscall.PROT_READ | syscall.PROT_WRITE
	flags := syscall.MAP_PRIVATE | syscall.MAP_ANON

	ptr, _, errno := syscall.Syscall6(
		syscall.SYS_MMAP,
		uintptr(addr),
		uintptr(length),
		uintptr(prot),
		uintptr(flags),
		^uintptr(0), // fd = -1
		0,           // offset = 0
	)

	if errno != 0 {
		return nil
	}
	return unsafe.Pointer(ptr)
}

// Vmfree 释放大块内存
func Vmfree(addr unsafe.Pointer, length int) {
	syscall.Syscall(
		syscall.SYS_MUNMAP,
		uintptr(addr),
		uintptr(length),
		0,
	)
}
