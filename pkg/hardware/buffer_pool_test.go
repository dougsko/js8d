package hardware

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestAudioBufferPool(t *testing.T) {
	pool := NewAudioBufferPool(16384, true)

	t.Run("Basic Buffer Operations", func(t *testing.T) {
		// Test small buffer
		buffer := pool.Get(1024)
		if buffer == nil {
			t.Fatal("Expected non-nil buffer")
		}
		if len(buffer.Data) != 1024 {
			t.Errorf("Expected buffer size 1024, got %d", len(buffer.Data))
		}
		if cap(buffer.Data) < 1024 {
			t.Errorf("Expected buffer capacity >= 1024, got %d", cap(buffer.Data))
		}

		// Test buffer recycling
		pool.Put(buffer)

		// Get another buffer - should reuse from pool
		buffer2 := pool.Get(1024)
		if buffer2 == nil {
			t.Fatal("Expected non-nil buffer")
		}
		if len(buffer2.Data) != 1024 {
			t.Errorf("Expected recycled buffer size 1024, got %d", len(buffer2.Data))
		}
	})

	t.Run("Different Buffer Sizes", func(t *testing.T) {
		// Test small pool (<=1024)
		small := pool.Get(512)
		if len(small.Data) != 512 {
			t.Errorf("Expected small buffer size 512, got %d", len(small.Data))
		}

		// Test medium pool (<=4096)
		medium := pool.Get(2048)
		if len(medium.Data) != 2048 {
			t.Errorf("Expected medium buffer size 2048, got %d", len(medium.Data))
		}

		// Test large pool (<=16384)
		large := pool.Get(8192)
		if len(large.Data) != 8192 {
			t.Errorf("Expected large buffer size 8192, got %d", len(large.Data))
		}

		// Clean up
		pool.Put(small)
		pool.Put(medium)
		pool.Put(large)
	})

	t.Run("Buffer Reset", func(t *testing.T) {
		buffer := pool.Get(100)

		// Fill with test data
		for i := range buffer.Data {
			buffer.Data[i] = int16(i + 1000)
		}

		// Reset should zero the buffer
		buffer.Reset()
		for i := range buffer.Data {
			if buffer.Data[i] != 0 {
				t.Errorf("Expected buffer[%d] to be 0 after reset, got %d", i, buffer.Data[i])
			}
		}

		pool.Put(buffer)
	})

	t.Run("Oversized Buffer", func(t *testing.T) {
		// Request buffer larger than max size
		oversized := pool.Get(20000)
		if oversized == nil {
			t.Fatal("Expected non-nil buffer even for oversized request")
		}
		if len(oversized.Data) != 20000 {
			t.Errorf("Expected oversized buffer size 20000, got %d", len(oversized.Data))
		}

		// This should not go back to pool due to size
		pool.Put(oversized)
	})

	t.Run("Invalid Size", func(t *testing.T) {
		// Test zero size
		buffer := pool.Get(0)
		if buffer == nil {
			t.Fatal("Expected non-nil buffer even for zero size")
		}

		// Test negative size
		buffer2 := pool.Get(-100)
		if buffer2 == nil {
			t.Fatal("Expected non-nil buffer even for negative size")
		}
	})
}

func TestGlobalAudioPool(t *testing.T) {
	t.Run("Singleton Behavior", func(t *testing.T) {
		pool1 := GetGlobalAudioPool()
		pool2 := GetGlobalAudioPool()

		if pool1 != pool2 {
			t.Error("Expected same pool instance from GetGlobalAudioPool()")
		}
	})

	t.Run("Convenience Functions", func(t *testing.T) {
		// Test GetAudioBuffer
		buffer := GetAudioBuffer(1024)
		if buffer == nil {
			t.Fatal("Expected non-nil buffer from GetAudioBuffer")
		}
		if len(buffer.Data) != 1024 {
			t.Errorf("Expected buffer size 1024, got %d", len(buffer.Data))
		}

		// Test buffer release
		buffer.Release()

		// Test GetAudioBufferSlice
		slice := GetAudioBufferSlice(512)
		if slice == nil {
			t.Fatal("Expected non-nil slice from GetAudioBufferSlice")
		}
		if len(slice) != 512 {
			t.Errorf("Expected slice size 512, got %d", len(slice))
		}

		// Test PutAudioBufferSlice
		PutAudioBufferSlice(slice)

		// Test RecycleAudioSamples
		samples := GetAudioBufferSlice(256)
		RecycleAudioSamples(samples)
	})
}

func TestAudioBufferPoolConcurrency(t *testing.T) {
	pool := NewAudioBufferPool(16384, true)

	t.Run("Concurrent Access", func(t *testing.T) {
		const numWorkers = 50
		const buffersPerWorker = 100

		var wg sync.WaitGroup
		wg.Add(numWorkers)

		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < buffersPerWorker; j++ {
					// Get buffer of varying sizes
					size := 500 + (workerID*10) + j
					if size > 16000 {
						size = 1000
					}

					buffer := pool.Get(size)
					if buffer == nil {
						t.Errorf("Worker %d: Got nil buffer for size %d", workerID, size)
						continue
					}

					if len(buffer.Data) != size {
						t.Errorf("Worker %d: Expected size %d, got %d", workerID, size, len(buffer.Data))
					}

					// Fill with test data
					for k := range buffer.Data {
						buffer.Data[k] = int16(workerID*1000 + j*10 + k)
					}

					// Simulate some work
					time.Sleep(time.Microsecond)

					// Return to pool
					pool.Put(buffer)
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestAudioBufferPoolStatistics(t *testing.T) {
	pool := NewAudioBufferPool(16384, true)

	t.Run("Statistics Tracking", func(t *testing.T) {
		// Get some buffers to generate statistics
		buffers := make([]*AudioBuffer, 10)
		for i := 0; i < 10; i++ {
			buffers[i] = pool.Get(1024)
		}

		// Return them to pool
		for _, buffer := range buffers {
			pool.Put(buffer)
		}

		// Get some more to generate hits
		for i := 0; i < 5; i++ {
			buffer := pool.Get(1024)
			pool.Put(buffer)
		}

		stats := pool.GetStatistics()
		if stats["small_hits"] < 5 {
			t.Errorf("Expected at least 5 small hits, got %d", stats["small_hits"])
		}

		if stats["small_miss"] < 10 {
			t.Errorf("Expected at least 10 small misses, got %d", stats["small_miss"])
		}
	})
}

func TestBufferPoolMemoryEfficiency(t *testing.T) {
	// This is more of a demonstration test showing memory usage patterns
	t.Run("Memory Usage Comparison", func(t *testing.T) {
		const numBuffers = 1000
		const bufferSize = 1024

		// Test without pool (traditional allocation)
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Allocate many buffers traditionally
		traditional := make([][]int16, numBuffers)
		for i := 0; i < numBuffers; i++ {
			traditional[i] = make([]int16, bufferSize)
		}

		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		traditionalAlloc := m2.Alloc - m1.Alloc

		// Test with pool
		runtime.GC()
		var m3 runtime.MemStats
		runtime.ReadMemStats(&m3)

		pool := NewAudioBufferPool(16384, true)
		pooled := make([]*AudioBuffer, numBuffers)
		for i := 0; i < numBuffers; i++ {
			pooled[i] = pool.Get(bufferSize)
		}

		// Return to pool
		for _, buffer := range pooled {
			pool.Put(buffer)
		}

		// Reuse from pool
		for i := 0; i < numBuffers; i++ {
			pooled[i] = pool.Get(bufferSize)
		}

		runtime.GC()
		var m4 runtime.MemStats
		runtime.ReadMemStats(&m4)

		pooledAlloc := m4.Alloc - m3.Alloc

		t.Logf("Traditional allocation: %d bytes", traditionalAlloc)
		t.Logf("Pooled allocation: %d bytes", pooledAlloc)
		t.Logf("Pool statistics: %+v", pool.GetStatistics())

		// Pool should generally use less memory due to reuse
		// Note: This is a rough comparison as GC timing affects results
		if pooledAlloc > traditionalAlloc*2 {
			t.Logf("Warning: Pool used significantly more memory than traditional allocation")
		}
	})
}

func BenchmarkAudioBufferPool(b *testing.B) {
	pool := NewAudioBufferPool(16384, false) // Disable stats for benchmarking

	b.Run("Get1024", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buffer := pool.Get(1024)
			pool.Put(buffer)
		}
	})

	b.Run("Traditional1024", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buffer := make([]int16, 1024)
			_ = buffer // Prevent optimization
		}
	})

	b.Run("GetMixed", func(b *testing.B) {
		sizes := []int{512, 1024, 2048, 4096}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			size := sizes[i%len(sizes)]
			buffer := pool.Get(size)
			pool.Put(buffer)
		}
	})

	b.Run("TraditionalMixed", func(b *testing.B) {
		sizes := []int{512, 1024, 2048, 4096}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			size := sizes[i%len(sizes)]
			buffer := make([]int16, size)
			_ = buffer // Prevent optimization
		}
	})
}