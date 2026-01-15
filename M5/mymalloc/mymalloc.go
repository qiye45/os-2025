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

// MallocCount 用于测试的计数器（可以移除）
var MallocCount int64

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
// TODO: 实现内存分配逻辑
func Mymalloc(size int) unsafe.Pointer {
	bigLock.Lock()
	atomic.AddInt64(&MallocCount, 1)
	bigLock.Unlock()

	// TODO: 实现内存分配
	return nil
}

// Myfree 释放之前分配的内存
// TODO: 实现内存释放逻辑
func Myfree(ptr unsafe.Pointer) {
	// TODO: 实现内存释放
}
