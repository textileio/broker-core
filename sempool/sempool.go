package sempool

import "sync"

// NewSemaphore returns a new semaphore with capacity.
func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{inner: make(chan struct{}, capacity)}
}

// Semaphore wraps a channel used as a semaphore.
type Semaphore struct {
	inner chan struct{}
}

// Acquire blocks while acquiring the semaphore.
func (s *Semaphore) Acquire() {
	s.inner <- struct{}{}
}

// TryAcquire tries to acquire the semaphore but will not block if it's unavailable.
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.inner <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release releass the semaphore.
func (s *Semaphore) Release() {
	select {
	case <-s.inner:
	default:
		panic("thread semaphore inconsistency: release before acquire!")
	}
}

// SemaphoreKey allows a custom types to define their own semaphore pool key.
type SemaphoreKey interface {
	Key() string
}

// NewSemaphorePool returns a new SemaphorePool.
func NewSemaphorePool(semaCap int) *SemaphorePool {
	return &SemaphorePool{ss: make(map[string]*Semaphore), semaCap: semaCap}
}

// SemaphorePool is a collection of semaphores with a common key pattern.
type SemaphorePool struct {
	ss      map[string]*Semaphore
	semaCap int
	mu      sync.Mutex
}

// Get returns a semaphore by key.
func (p *SemaphorePool) Get(k SemaphoreKey) *Semaphore {
	var (
		s     *Semaphore
		exist bool
		key   = k.Key()
	)

	p.mu.Lock()
	if s, exist = p.ss[key]; !exist {
		s = NewSemaphore(p.semaCap)
		p.ss[key] = s
	}
	p.mu.Unlock()

	return s
}

// Stop stops the pool by blocking while acquiring all semaphores.
func (p *SemaphorePool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// grab all semaphores and hold
	for _, s := range p.ss {
		s.Acquire()
	}
}
