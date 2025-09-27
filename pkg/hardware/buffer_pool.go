package hardware

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// AudioBuffer represents a reusable audio buffer with metadata
type AudioBuffer struct {
	Data []int16
	Size int
	pool *AudioBufferPool
}

// Reset clears the buffer data and resets size for reuse
func (ab *AudioBuffer) Reset() {
	// Zero out the buffer to prevent data leakage
	for i := range ab.Data {
		ab.Data[i] = 0
	}
	ab.Size = 0
}

// Release returns the buffer to its pool for reuse
func (ab *AudioBuffer) Release() {
	if ab.pool != nil {
		ab.pool.Put(ab)
	}
}

// AudioBufferPool manages pools of audio buffers for different size ranges
type AudioBufferPool struct {
	// Size-based pools for optimal memory reuse
	smallPool  *sync.Pool // <= 1024 samples
	mediumPool *sync.Pool // <= 4096 samples
	largePool  *sync.Pool // <= 16384 samples

	// Pool statistics for monitoring and optimization
	smallHits   int64
	mediumHits  int64
	largeHits   int64
	smallMiss   int64
	mediumMiss  int64
	largeMiss   int64

	// Configuration
	maxBufferSize    int
	enableStatistics bool
	statsMutex       sync.RWMutex
}

// Global audio buffer pool instance
var globalAudioPool *AudioBufferPool
var poolOnce sync.Once

// GetGlobalAudioPool returns the singleton audio buffer pool
func GetGlobalAudioPool() *AudioBufferPool {
	poolOnce.Do(func() {
		globalAudioPool = NewAudioBufferPool(16384, true)

		// Start periodic statistics logging
		go globalAudioPool.statisticsReporter()
	})
	return globalAudioPool
}

// NewAudioBufferPool creates a new audio buffer pool with size-based sub-pools
func NewAudioBufferPool(maxBufferSize int, enableStats bool) *AudioBufferPool {
	pool := &AudioBufferPool{
		maxBufferSize:    maxBufferSize,
		enableStatistics: enableStats,
	}

	// Initialize small buffer pool (up to 1024 samples)
	pool.smallPool = &sync.Pool{
		New: func() interface{} {
			if enableStats {
				atomic.AddInt64(&pool.smallMiss, 1)
			}
			return &AudioBuffer{
				Data: make([]int16, 1024),
				pool: pool,
			}
		},
	}

	// Initialize medium buffer pool (up to 4096 samples)
	pool.mediumPool = &sync.Pool{
		New: func() interface{} {
			if enableStats {
				atomic.AddInt64(&pool.mediumMiss, 1)
			}
			return &AudioBuffer{
				Data: make([]int16, 4096),
				pool: pool,
			}
		},
	}

	// Initialize large buffer pool (up to 16384 samples)
	pool.largePool = &sync.Pool{
		New: func() interface{} {
			if enableStats {
				atomic.AddInt64(&pool.largeMiss, 1)
			}
			return &AudioBuffer{
				Data: make([]int16, 16384),
				pool: pool,
			}
		},
	}

	return pool
}

// Get retrieves a buffer of at least the requested size from the appropriate pool
func (p *AudioBufferPool) Get(size int) *AudioBuffer {
	if size <= 0 {
		log.Printf("AudioBufferPool: Invalid buffer size requested: %d", size)
		return &AudioBuffer{
			Data: make([]int16, 1024),
			Size: size,
			pool: p,
		}
	}

	if size > p.maxBufferSize {
		log.Printf("AudioBufferPool: Requested size %d exceeds max %d, allocating directly",
			size, p.maxBufferSize)
		return &AudioBuffer{
			Data: make([]int16, size),
			Size: size,
			pool: p,
		}
	}

	var buffer *AudioBuffer

	// Select appropriate pool based on size
	switch {
	case size <= 1024:
		buffer = p.smallPool.Get().(*AudioBuffer)
		if p.enableStatistics {
			atomic.AddInt64(&p.smallHits, 1)
		}
	case size <= 4096:
		buffer = p.mediumPool.Get().(*AudioBuffer)
		if p.enableStatistics {
			atomic.AddInt64(&p.mediumHits, 1)
		}
	default:
		buffer = p.largePool.Get().(*AudioBuffer)
		if p.enableStatistics {
			atomic.AddInt64(&p.largeHits, 1)
		}
	}

	// Ensure buffer is large enough and set actual size
	if cap(buffer.Data) < size {
		// This shouldn't happen with our pool design, but handle gracefully
		log.Printf("AudioBufferPool: Pool buffer too small (cap=%d, need=%d), reallocating",
			cap(buffer.Data), size)
		buffer.Data = make([]int16, size)
	}

	// Set the slice length to the requested size
	buffer.Data = buffer.Data[:size]
	buffer.Size = size

	return buffer
}

// Put returns a buffer to the appropriate pool for reuse
func (p *AudioBufferPool) Put(buffer *AudioBuffer) {
	if buffer == nil || buffer.Data == nil {
		return
	}

	// Reset buffer data to prevent leakage
	buffer.Reset()

	// Determine which pool to return the buffer to based on capacity
	capacity := cap(buffer.Data)

	switch {
	case capacity <= 1024:
		p.smallPool.Put(buffer)
	case capacity <= 4096:
		p.mediumPool.Put(buffer)
	case capacity <= 16384:
		p.largePool.Put(buffer)
	default:
		// Don't return oversized buffers to pool to prevent memory bloat
		// They will be garbage collected
	}
}

// GetBufferSlice is a convenience method that returns just the []int16 slice
// WARNING: Caller must manually call PutBufferSlice when done!
func (p *AudioBufferPool) GetBufferSlice(size int) []int16 {
	buffer := p.Get(size)
	// Store the buffer reference in the slice for later retrieval
	// This is a hack but allows backward compatibility
	return buffer.Data
}

// PutBufferSlice returns a slice to the pool (requires the AudioBuffer metadata)
// This is less safe than using AudioBuffer.Release() directly
func (p *AudioBufferPool) PutBufferSlice(slice []int16) {
	// This is tricky - we need to reconstruct the AudioBuffer
	// For now, we'll create a temporary one and put it back
	buffer := &AudioBuffer{
		Data: slice,
		Size: len(slice),
		pool: p,
	}
	p.Put(buffer)
}

// GetStatistics returns current pool utilization statistics
func (p *AudioBufferPool) GetStatistics() map[string]int64 {
	if !p.enableStatistics {
		return map[string]int64{}
	}

	return map[string]int64{
		"small_hits":   atomic.LoadInt64(&p.smallHits),
		"medium_hits":  atomic.LoadInt64(&p.mediumHits),
		"large_hits":   atomic.LoadInt64(&p.largeHits),
		"small_miss":   atomic.LoadInt64(&p.smallMiss),
		"medium_miss":  atomic.LoadInt64(&p.mediumMiss),
		"large_miss":   atomic.LoadInt64(&p.largeMiss),
	}
}

// statisticsReporter periodically logs pool statistics
func (p *AudioBufferPool) statisticsReporter() {
	if !p.enableStatistics {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := p.GetStatistics()

		totalHits := stats["small_hits"] + stats["medium_hits"] + stats["large_hits"]
		totalMiss := stats["small_miss"] + stats["medium_miss"] + stats["large_miss"]
		totalRequests := totalHits + totalMiss

		if totalRequests > 0 {
			hitRate := float64(totalHits) / float64(totalRequests) * 100
			log.Printf("AudioBufferPool Stats: %d requests, %.1f%% hit rate (S:%d/%d M:%d/%d L:%d/%d)",
				totalRequests, hitRate,
				stats["small_hits"], stats["small_miss"],
				stats["medium_hits"], stats["medium_miss"],
				stats["large_hits"], stats["large_miss"])
		}
	}
}

// Convenience functions for global pool access

// GetAudioBuffer gets a buffer from the global pool
func GetAudioBuffer(size int) *AudioBuffer {
	return GetGlobalAudioPool().Get(size)
}

// GetAudioBufferSlice gets a buffer slice from the global pool
func GetAudioBufferSlice(size int) []int16 {
	return GetGlobalAudioPool().GetBufferSlice(size)
}

// PutAudioBufferSlice returns a buffer slice to the global pool
func PutAudioBufferSlice(slice []int16) {
	GetGlobalAudioPool().PutBufferSlice(slice)
}

// RecycleAudioSamples can be called by consumers to optionally recycle audio sample buffers
// This is a performance optimization - calling this is optional but recommended
func RecycleAudioSamples(samples []int16) {
	if len(samples) > 0 {
		PutAudioBufferSlice(samples)
	}
}