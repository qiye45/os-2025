package mymalloc

import (
	"sync/atomic"
	"syscall"
	"unsafe"
)

// SpinLock 自旋锁实现
type SpinLock struct {
	status int32
}

const (
	locked   = 1
	unlocked = 0
	pageSize = 4096
)

// Lock 获取自旋锁
func (s *SpinLock) Lock() {
	for !atomic.CompareAndSwapInt32(&s.status, unlocked, locked) {
		// 自旋等待
	}
}

// Unlock 释放自旋锁
func (s *SpinLock) Unlock() {
	atomic.StoreInt32(&s.status, unlocked)
}

// 全局大锁
var bigLock SpinLock

// MallocCount 用于测试的计数器
var MallocCount int64

// 记录指针到size的映射
var ptrMap map[unsafe.Pointer]int
var pool MemoryPool
var sizeClass []int

type MemoryPool struct {
	page     unsafe.Pointer
	offset   int
	freeList [][]unsafe.Pointer
}

func (m *MemoryPool) malloc(size int) unsafe.Pointer {
	//	大块直接mmap
	if size > sizeClass[len(sizeClass)-1] {
		pages := (size + pageSize - 1) / pageSize
		p := Vmalloc(nil, pages*pageSize)
		ptrMap[p] = pages * pageSize
		return p
	}
	// 从空闲链表查找，保证空闲块的大小是>=申请的size的块
	fixSize, sizeIndex := getFixSize(size)
	if len(m.freeList[sizeIndex]) > 0 {
		idx := len(m.freeList[sizeIndex]) - 1
		p := m.freeList[sizeIndex][idx]
		m.freeList[sizeIndex] = m.freeList[sizeIndex][:idx]
		ptrMap[p] = fixSize
		return p
	}
	// 从当前页分配
	if m.page == nil || m.offset+fixSize > pageSize {
		m.page = Vmalloc(nil, pageSize)
		m.offset = 0
	}
	p := unsafe.Pointer(uintptr(m.page) + uintptr(m.offset))
	ptrMap[p] = fixSize
	m.offset += fixSize
	return p
}

func (m *MemoryPool) free(p unsafe.Pointer) {
	// 大块直接释放
	if size := ptrMap[p]; size > sizeClass[len(sizeClass)-1] {
		Vmfree(p, size) // 申请多少，释放多少
		delete(ptrMap, p)
		return
	}

	// 小块添加到空闲链表
	_, sizeIndex := getFixSize(ptrMap[p])
	delete(ptrMap, p) // 删除指针映射，避免内存泄漏
	m.freeList[sizeIndex] = append(m.freeList[sizeIndex], p)
	// 为什么不清空使用过的内存？
	//✅ 调用者的责任是使用前初始化
	//✅ 避免无意义的清零操作
	//✅ 这是工业级分配器的标准做法
}

func init() {
	ptrMap = make(map[unsafe.Pointer]int)
	pool = MemoryPool{
		page:   nil,
		offset: 0,
	}
	sizeClass = []int{8, 16, 32, 64}
	for _ = range sizeClass {
		pool.freeList = append(pool.freeList, []unsafe.Pointer{})
	}
}

func getFixSize(size int) (int, int) {
	for i, class := range sizeClass {
		if size <= class {
			return class, i
		}
	}
	return sizeClass[len(sizeClass)-1], len(sizeClass) - 1
}

// Reset 重置分配器状态（专门用于测试隔离）
// 注意：这不会释放已经分配出去但未归还的内存，仅重置内部状态
func Reset() {
	bigLock.Lock()
	defer bigLock.Unlock()

	// 重置当前页指针，强制下次分配申请新页
	pool.page = nil
	pool.offset = 0

	// 清空空闲链表，丢弃之前的碎片
	for i := range pool.freeList {
		pool.freeList[i] = pool.freeList[i][:0]
	}

	// 注意：我们不清空 ptrMap，以免影响未被测试逻辑覆盖的指针
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

// Mymalloc 分配 size 字节的内存
// 返回 8 字节对齐的内存地址，失败返回 nil
func Mymalloc(size int) unsafe.Pointer {
	bigLock.Lock()
	defer bigLock.Unlock()

	atomic.AddInt64(&MallocCount, 1)

	if size <= 0 {
		return nil
	}

	// 实现内存分配
	//p := Vmalloc(nil, size)
	//ptrMap[p] = size
	// memory waste 512.00x  -> 1.02x

	p := pool.malloc(size)
	return p
}

// Myfree 释放之前分配的内存
func Myfree(ptr unsafe.Pointer) {
	// 实现内存释放
	bigLock.Lock()
	defer bigLock.Unlock()

	//size := ptrMap[ptr]
	//Vmfree(ptr, size)
	//delete(ptrMap, ptr)

	pool.free(ptr)
}
