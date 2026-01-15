package mymalloc

import (
	"sync"
	"sync/atomic"
	"testing"
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

const N = 100000

func tMalloc() {
	for i := 0; i < N; i++ {
		Mymalloc(0)
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
