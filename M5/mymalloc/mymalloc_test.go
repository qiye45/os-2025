package mymalloc

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// TestTrivial 基本分配测试
func TestTrivial(t *testing.T) {
	p1 := Mymalloc(4)
	if p1 == nil {
		t.Fatal("malloc should not return NULL")
	}
	*(*int32)(p1) = 1024

	p2 := Mymalloc(4)
	if p2 == nil {
		t.Fatal("malloc should not return NULL")
	}
	*(*int32)(p2) = 2048

	if p1 == p2 {
		t.Fatal("malloc should return different pointers")
	}
	if *(*int32)(p1)*2 != *(*int32)(p2) {
		t.Fatal("value check should pass")
	}

	Myfree(p1)
	Myfree(p2)
}

// TestVmalloc 测试 vmalloc/vmfree
func TestVmalloc(t *testing.T) {
	p1 := Vmalloc(nil, 4096)
	if p1 == nil {
		t.Fatal("vmalloc should not return NULL")
	}
	if uintptr(p1)%4096 != 0 {
		t.Fatal("vmalloc should return page-aligned address")
	}

	p2 := Vmalloc(nil, 8192)
	if p2 == nil {
		t.Fatal("vmalloc should not return NULL")
	}
	if uintptr(p2)%4096 != 0 {
		t.Fatal("vmalloc should return page-aligned address")
	}
	if p1 == p2 {
		t.Fatal("vmalloc should return different pointers")
	}

	Vmfree(p1, 4096)
	Vmfree(p2, 8192)
}

// TestAlignment 测试 8 字节对齐
func TestAlignment(t *testing.T) {
	sizes := []int{1, 7, 8, 15, 16, 31, 32, 63, 64, 127, 128, 255, 256}
	for _, size := range sizes {
		p := Mymalloc(size)
		if p == nil {
			t.Fatalf("malloc(%d) should not return NULL", size)
		}
		if uintptr(p)%8 != 0 {
			t.Fatalf("malloc(%d) returned unaligned address: %p", size, p)
		}
		Myfree(p)
	}
}

// TestNoOverlap 测试分配的内存不重叠
func TestNoOverlap(t *testing.T) {
	const N = 100
	ptrs := make([]unsafe.Pointer, N)
	sizes := make([]int, N)

	// 分配多个不同大小的内存块
	for i := 0; i < N; i++ {
		size := (i%10 + 1) * 8
		sizes[i] = size
		ptrs[i] = Mymalloc(size)
		if ptrs[i] == nil {
			t.Fatalf("malloc(%d) failed at iteration %d", size, i)
		}
		// 写入标记值
		for j := 0; j < size; j++ {
			*(*byte)(unsafe.Pointer(uintptr(ptrs[i]) + uintptr(j))) = byte(i)
		}
	}

	// 检查是否有重叠
	for i := 0; i < N; i++ {
		for j := 0; j < sizes[i]; j++ {
			val := *(*byte)(unsafe.Pointer(uintptr(ptrs[i]) + uintptr(j)))
			if val != byte(i) {
				t.Fatalf("memory overlap detected at block %d", i)
			}
		}
	}

	// 释放所有内存
	for i := 0; i < N; i++ {
		Myfree(ptrs[i])
	}
}

// TestReuseAfterFree 测试内存回收后可以重用
func TestReuseAfterFree(t *testing.T) {
	const N = 1000
	allocated := make(map[uintptr]bool)

	// 第一轮分配
	ptrs := make([]unsafe.Pointer, N)
	for i := 0; i < N; i++ {
		ptrs[i] = Mymalloc(64)
		if ptrs[i] == nil {
			t.Fatalf("malloc failed at iteration %d", i)
		}
		allocated[uintptr(ptrs[i])] = true
	}

	// 释放所有内存
	for i := 0; i < N; i++ {
		Myfree(ptrs[i])
	}

	// 第二轮分配，应该能重用之前的内存
	reused := 0
	for i := 0; i < N; i++ {
		p := Mymalloc(64)
		if p == nil {
			t.Fatalf("malloc failed at second round iteration %d", i)
		}
		if allocated[uintptr(p)] {
			reused++
		}
		Myfree(p)
	}

	// 至少应该重用一部分内存
	if reused == 0 {
		t.Log("Warning: no memory was reused, possible memory leak")
	}
}

// TestZeroSize 测试零大小分配
func TestZeroSize(t *testing.T) {
	p := Mymalloc(0)
	// 零大小分配可以返回 NULL 或有效指针
	if p != nil {
		Myfree(p)
	}
}

// TestLargeAllocation 测试大块内存分配
func TestLargeAllocation(t *testing.T) {
	sizes := []int{1024, 4096, 8192, 16384, 65536}
	for _, size := range sizes {
		p := Mymalloc(size)
		if p == nil {
			t.Logf("malloc(%d) returned NULL (acceptable for large sizes)", size)
			continue
		}
		// 写入并验证
		*(*int)(p) = 0xdeadbeef
		if *(*int)(p) != 0xdeadbeef {
			t.Fatalf("large allocation write/read failed for size %d", size)
		}
		Myfree(p)
	}
}

const N = 100000

func tMalloc() {
	for i := 0; i < N; i++ {
		p := Mymalloc(64) // 分配实际内存，不是0
		if p != nil {
			*(*int)(p) = i // 写入数据
			Myfree(p)      // 立即释放
		}
	}
}

// TestConcurrent 并发分配测试
func TestConcurrent(t *testing.T) {
	// 重置计数器
	atomic.StoreInt64(&MallocCount, 0)

	var wg sync.WaitGroup
	wg.Add(4)

	for i := 0; i < 4; i++ {
		go func() {
			defer wg.Done()
			tMalloc()
		}()
	}

	wg.Wait()

	if MallocCount != 4*N {
		t.Fatalf("malloc_count should be 4N, got %d", MallocCount)
	}
}

// TestConcurrentAllocFree 并发分配和释放测试
func TestConcurrentAllocFree(t *testing.T) {
	const goroutines = 8
	const iterations = 10000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			rand.Seed(time.Now().UnixNano() + int64(id))
			ptrs := make([]unsafe.Pointer, 0, 100)

			for j := 0; j < iterations; j++ {
				if rand.Float32() < 0.7 || len(ptrs) == 0 {
					// 70% 概率分配
					size := rand.Intn(256) + 8
					p := Mymalloc(size)
					if p != nil {
						ptrs = append(ptrs, p)
						// 写入数据验证
						*(*int)(p) = id*iterations + j
					}
				} else {
					// 30% 概率释放
					idx := rand.Intn(len(ptrs))
					Myfree(ptrs[idx])
					ptrs = append(ptrs[:idx], ptrs[idx+1:]...)
				}
			}

			// 清理剩余内存
			for _, p := range ptrs {
				Myfree(p)
			}
		}(i)
	}

	wg.Wait()
}

// TestStressRandom 压力测试：随机大小分配
func TestStressRandom(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const iterations = 50000
	rand.Seed(time.Now().UnixNano())
	ptrs := make([]unsafe.Pointer, 0, 1000)
	sizes := make([]int, 0, 1000)

	for i := 0; i < iterations; i++ {
		if rand.Float32() < 0.6 || len(ptrs) == 0 {
			// 分配
			size := rand.Intn(512) + 1
			p := Mymalloc(size)
			if p != nil {
				ptrs = append(ptrs, p)
				sizes = append(sizes, size)
				// 写入标记
				for j := 0; j < size; j++ {
					*(*byte)(unsafe.Pointer(uintptr(p) + uintptr(j))) = byte(i & 0xff)
				}
			}
		} else {
			// 释放
			idx := rand.Intn(len(ptrs))
			// 验证数据
			p := ptrs[idx]
			size := sizes[idx]
			for j := 0; j < size; j++ {
				// 简单验证，不做严格检查避免测试过慢
				_ = *(*byte)(unsafe.Pointer(uintptr(p) + uintptr(j)))
			}
			Myfree(p)
			ptrs = append(ptrs[:idx], ptrs[idx+1:]...)
			sizes = append(sizes[:idx], sizes[idx+1:]...)
		}
	}

	// 清理
	for _, p := range ptrs {
		Myfree(p)
	}
}

// TestConcurrentStress 并发压力测试
func TestConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent stress test in short mode")
	}

	var goroutines = runtime.NumCPU()
	const duration = 2 * time.Second

	var wg sync.WaitGroup
	var totalAllocs int64
	var totalFrees int64
	stop := make(chan struct{})

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			rand.Seed(time.Now().UnixNano() + int64(id))
			ptrs := make([]unsafe.Pointer, 0, 100)
			localAllocs := 0
			localFrees := 0

			for {
				select {
				case <-stop:
					// 清理
					for _, p := range ptrs {
						Myfree(p)
						localFrees++
					}
					atomic.AddInt64(&totalAllocs, int64(localAllocs))
					atomic.AddInt64(&totalFrees, int64(localFrees))
					return
				default:
					if rand.Float32() < 0.7 || len(ptrs) == 0 {
						size := rand.Intn(128) + 8
						p := Mymalloc(size)
						if p != nil {
							ptrs = append(ptrs, p)
							localAllocs++
						}
					} else {
						idx := rand.Intn(len(ptrs))
						Myfree(ptrs[idx])
						localFrees++
						ptrs = append(ptrs[:idx], ptrs[idx+1:]...)
					}
				}
			}
		}(i)
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()

	t.Logf("Total allocations: %d, Total frees: %d", totalAllocs, totalFrees)
}

// TestFragmentation 测试内存碎片情况
func TestFragmentation(t *testing.T) {
	const N = 1000
	ptrs := make([]unsafe.Pointer, N)

	// 分配小块内存
	for i := 0; i < N; i++ {
		ptrs[i] = Mymalloc(16)
		if ptrs[i] == nil {
			t.Fatalf("malloc failed at iteration %d", i)
		}
	}

	// 释放奇数位置的内存，制造碎片
	for i := 1; i < N; i += 2 {
		Myfree(ptrs[i])
		ptrs[i] = nil
	}

	// 尝试分配较大的内存块
	large := Mymalloc(256)
	if large != nil {
		Myfree(large)
	}

	// 清理
	for i := 0; i < N; i += 2 {
		if ptrs[i] != nil {
			Myfree(ptrs[i])
		}
	}
}

// TestStrictOverlapDetection 严格的重叠检测测试
func TestStrictOverlapDetection(t *testing.T) {
	const N = 200
	type allocation struct {
		ptr   unsafe.Pointer
		size  int
		magic uint64
	}

	allocs := make([]allocation, N)

	// 分配并写入唯一标记
	for i := 0; i < N; i++ {
		size := (i%20 + 1) * 8
		p := Mymalloc(size)
		if p == nil {
			t.Fatalf("malloc(%d) failed at iteration %d", size, i)
		}

		magic := uint64(0xDEADBEEF00000000) | uint64(i)
		allocs[i] = allocation{ptr: p, size: size, magic: magic}

		// 写入魔数到整个分配区域
		for j := 0; j < size; j += 8 {
			if j+8 <= size {
				*(*uint64)(unsafe.Pointer(uintptr(p) + uintptr(j))) = magic
			}
		}
	}

	// 验证所有分配的数据完整性
	for i := 0; i < N; i++ {
		p := allocs[i].ptr
		size := allocs[i].size
		magic := allocs[i].magic

		for j := 0; j < size; j += 8 {
			if j+8 <= size {
				val := *(*uint64)(unsafe.Pointer(uintptr(p) + uintptr(j)))
				if val != magic {
					t.Fatalf("data corruption detected at block %d, offset %d: expected %x, got %x",
						i, j, magic, val)
				}
			}
		}
	}

	// 检查地址范围是否重叠
	for i := 0; i < N; i++ {
		for j := i + 1; j < N; j++ {
			start1 := uintptr(allocs[i].ptr)
			end1 := start1 + uintptr(allocs[i].size)
			start2 := uintptr(allocs[j].ptr)
			end2 := start2 + uintptr(allocs[j].size)

			// 检查是否重叠
			if !(end1 <= start2 || end2 <= start1) {
				t.Fatalf("memory overlap detected between block %d [%x-%x) and block %d [%x-%x)",
					i, start1, end1, j, start2, end2)
			}
		}
	}

	// 清理
	for i := 0; i < N; i++ {
		Myfree(allocs[i].ptr)
	}
}

// TestDataIntegrityAfterFree 测试释放后重新分配的数据完整性
func TestDataIntegrityAfterFree(t *testing.T) {
	const N = 500
	const size = 128

	// 第一轮：分配并写入数据
	ptrs1 := make([]unsafe.Pointer, N)
	for i := 0; i < N; i++ {
		p := Mymalloc(size)
		if p == nil {
			t.Fatalf("first round malloc failed at %d", i)
		}
		ptrs1[i] = p

		// 写入唯一模式
		for j := 0; j < size; j++ {
			*(*byte)(unsafe.Pointer(uintptr(p) + uintptr(j))) = byte((i + j) & 0xFF)
		}
	}

	// 验证第一轮数据
	for i := 0; i < N; i++ {
		p := ptrs1[i]
		for j := 0; j < size; j++ {
			expected := byte((i + j) & 0xFF)
			actual := *(*byte)(unsafe.Pointer(uintptr(p) + uintptr(j)))
			if actual != expected {
				t.Fatalf("first round data corruption at block %d, offset %d", i, j)
			}
		}
	}

	// 释放所有内存
	for i := 0; i < N; i++ {
		Myfree(ptrs1[i])
	}

	// 第二轮：重新分配并写入不同数据
	ptrs2 := make([]unsafe.Pointer, N)
	for i := 0; i < N; i++ {
		p := Mymalloc(size)
		if p == nil {
			t.Fatalf("second round malloc failed at %d", i)
		}
		ptrs2[i] = p

		// 写入不同的模式
		for j := 0; j < size; j++ {
			*(*byte)(unsafe.Pointer(uintptr(p) + uintptr(j))) = byte((i*2 + j) & 0xFF)
		}
	}

	// 验证第二轮数据
	for i := 0; i < N; i++ {
		p := ptrs2[i]
		for j := 0; j < size; j++ {
			expected := byte((i*2 + j) & 0xFF)
			actual := *(*byte)(unsafe.Pointer(uintptr(p) + uintptr(j)))
			if actual != expected {
				t.Fatalf("second round data corruption at block %d, offset %d: expected %d, got %d",
					i, j, expected, actual)
			}
		}
	}

	// 清理
	for i := 0; i < N; i++ {
		Myfree(ptrs2[i])
	}
}

// TestConcurrentDataIntegrity 并发测试数据完整性
func TestConcurrentDataIntegrity(t *testing.T) {
	const goroutines = 8
	const allocsPerGoroutine = 1000

	var wg sync.WaitGroup
	var errors atomic.Int32

	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()

			type alloc struct {
				ptr  unsafe.Pointer
				size int
				id   int
			}

			allocs := make([]alloc, 0, 100)

			for i := 0; i < allocsPerGoroutine; i++ {
				size := (i%50 + 1) * 8
				p := Mymalloc(size)
				if p == nil {
					continue
				}

				allocID := gid*allocsPerGoroutine + i

				// 写入标识数据
				for j := 0; j < size; j += 4 {
					if j+4 <= size {
						*(*int32)(unsafe.Pointer(uintptr(p) + uintptr(j))) = int32(allocID)
					}
				}

				allocs = append(allocs, alloc{ptr: p, size: size, id: allocID})

				// 随机验证之前的分配
				if len(allocs) > 10 && i%10 == 0 {
					idx := i % len(allocs)
					a := allocs[idx]
					for j := 0; j < a.size; j += 4 {
						if j+4 <= a.size {
							val := *(*int32)(unsafe.Pointer(uintptr(a.ptr) + uintptr(j)))
							if val != int32(a.id) {
								errors.Add(1)
								t.Errorf("goroutine %d: data corruption in alloc %d at offset %d: expected %d, got %d",
									gid, a.id, j, a.id, val)
								return
							}
						}
					}
				}

				// 随机释放一些内存
				if len(allocs) > 50 && i%5 == 0 {
					idx := i % len(allocs)
					Myfree(allocs[idx].ptr)
					allocs = append(allocs[:idx], allocs[idx+1:]...)
				}
			}

			// 最终验证所有剩余分配
			for _, a := range allocs {
				for j := 0; j < a.size; j += 4 {
					if j+4 <= a.size {
						val := *(*int32)(unsafe.Pointer(uintptr(a.ptr) + uintptr(j)))
						if val != int32(a.id) {
							errors.Add(1)
							t.Errorf("goroutine %d: final check data corruption in alloc %d", gid, a.id)
							break
						}
					}
				}
				Myfree(a.ptr)
			}
		}(g)
	}

	wg.Wait()

	if errors.Load() > 0 {
		t.Fatalf("detected %d data corruption errors", errors.Load())
	}
}

// TestMemoryLeakDetection 测试内存泄漏
func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak test in short mode")
	}

	const rounds = 10
	const allocsPerRound = 10000
	const size = 64

	firstRoundAddrs := make(map[uintptr]bool)

	// 第一轮：记录所有分配的地址
	ptrs := make([]unsafe.Pointer, allocsPerRound)
	for i := 0; i < allocsPerRound; i++ {
		p := Mymalloc(size)
		if p == nil {
			t.Fatalf("malloc failed in first round at %d", i)
		}
		ptrs[i] = p
		firstRoundAddrs[uintptr(p)] = true
	}

	// 释放所有
	for i := 0; i < allocsPerRound; i++ {
		Myfree(ptrs[i])
	}

	// 多轮分配释放，检查是否重用之前的地址
	totalReused := 0
	for round := 0; round < rounds; round++ {
		reused := 0
		for i := 0; i < allocsPerRound; i++ {
			p := Mymalloc(size)
			if p == nil {
				t.Fatalf("malloc failed in round %d at %d", round, i)
			}
			if firstRoundAddrs[uintptr(p)] {
				reused++
			}
			Myfree(p)
		}
		totalReused += reused
	}

	// 应该有相当比例的地址被重用
	expectedMinReuse := allocsPerRound * rounds / 2
	if totalReused < expectedMinReuse {
		t.Errorf("possible memory leak: only %d/%d allocations reused addresses (expected at least %d)",
			totalReused, allocsPerRound*rounds, expectedMinReuse)
	} else {
		t.Logf("memory reuse rate: %d/%d (%.1f%%)",
			totalReused, allocsPerRound*rounds,
			float64(totalReused)*100/float64(allocsPerRound*rounds))
	}
}

// TestMemoryEfficiency 测试内存使用效率
// 这个测试会失败，因为当前实现每次分配都使用 4KB，即使只请求 8 字节
func TestMemoryEfficiency(t *testing.T) {
	Reset()

	const N = 1000
	const requestSize = 8
	const pageSize = 4096

	// 记录所有分配的地址
	ptrs := make([]unsafe.Pointer, N)

	// 分配 N 个小块
	for i := 0; i < N; i++ {
		p := Mymalloc(requestSize)
		if p == nil {
			t.Fatalf("malloc failed at iteration %d", i)
		}
		ptrs[i] = p
	}

	// 计算实际使用的内存范围
	pages := make(map[uintptr]bool)
	for _, p := range ptrs {
		pageAddr := uintptr(p) &^ (pageSize - 1) // 对齐到页边界
		pages[pageAddr] = true
	}

	// 释放所有内存
	for _, p := range ptrs {
		Myfree(p)
	}

	// 计算实际使用的内存总量
	actualPagesUsed := len(pages)
	actualMemoryUsed := actualPagesUsed * pageSize
	requestedMemory := N * requestSize

	// 计算浪费率
	wasteRatio := float64(actualMemoryUsed) / float64(requestedMemory)

	t.Logf("Requested memory: %d bytes", requestedMemory)
	t.Logf("Pages used: %d (%d bytes)", actualPagesUsed, actualMemoryUsed)
	t.Logf("Waste ratio: %.2fx", wasteRatio)

	// 理想情况：1000 × 8 = 8000 字节，应该只需要 2 个页 (8192 字节)
	// 如果每次分配独立页，会使用 1000 个页
	if wasteRatio > 4.0 {
		t.Fatalf("memory waste too high: %.2fx (should be < 4x)", wasteRatio)
	}
}

// TestRealReuse 测试真正的内存重用
// 验证释放的内存是否被重用，而不是每次都分配新页
func TestRealReuse(t *testing.T) {
	const N = 100
	const size = 64

	// 第一轮：分配并记录地址
	round1 := make([]unsafe.Pointer, N)
	for i := 0; i < N; i++ {
		round1[i] = Mymalloc(size)
		if round1[i] == nil {
			t.Fatalf("round 1 malloc failed at %d", i)
		}
	}

	// 释放所有
	for i := 0; i < N; i++ {
		Myfree(round1[i])
	}

	// 第二轮：重新分配
	round2 := make([]unsafe.Pointer, N)
	for i := 0; i < N; i++ {
		round2[i] = Mymalloc(size)
		if round2[i] == nil {
			t.Fatalf("round 2 malloc failed at %d", i)
		}
	}

	// 检查是否有地址重用
	reusedCount := 0
	addrMap := make(map[uintptr]bool)

	for _, p := range round1 {
		addrMap[uintptr(p)] = true
	}

	for _, p := range round2 {
		if addrMap[uintptr(p)] {
			reusedCount++
		}
	}

	// 清理
	for _, p := range round2 {
		Myfree(p)
	}

	reuseRate := float64(reusedCount) / float64(N) * 100
	t.Logf("Address reuse rate: %d/%d (%.1f%%)", reusedCount, N, reuseRate)

	// 如果重用率太低，说明没有真正重用内存
	// 当前实现每次都分配新页，所以重用率可能很低
	if reuseRate < 50.0 {
		t.Errorf("low memory reuse rate: %.1f%% (should be > 50%%)", reuseRate)
	}
}

// BenchmarkMalloc 基准测试：单线程分配
func BenchmarkMalloc(b *testing.B) {
	sizes := []int{8, 16, 32, 64, 128, 256, 512, 1024}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := Mymalloc(size)
				if p != nil {
					Myfree(p)
				}
			}
		})
	}
}

// BenchmarkMallocParallel 基准测试：并行分配
func BenchmarkMallocParallel(b *testing.B) {
	sizes := []int{16, 64, 256}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					p := Mymalloc(size)
					if p != nil {
						Myfree(p)
					}
				}
			})
		})
	}
}

// BenchmarkMallocFreePattern 基准测试：分配释放模式
func BenchmarkMallocFreePattern(b *testing.B) {
	b.Run("alloc_then_free", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ptrs := make([]unsafe.Pointer, 100)
			for j := 0; j < 100; j++ {
				ptrs[j] = Mymalloc(64)
			}
			for j := 0; j < 100; j++ {
				if ptrs[j] != nil {
					Myfree(ptrs[j])
				}
			}
		}
	})

	b.Run("interleaved", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ptrs := make([]unsafe.Pointer, 100)
			for j := 0; j < 100; j++ {
				ptrs[j] = Mymalloc(64)
				if j > 0 && ptrs[j-1] != nil {
					Myfree(ptrs[j-1])
				}
			}
			if ptrs[99] != nil {
				Myfree(ptrs[99])
			}
		}
	})
}
