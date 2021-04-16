package sempool

import "sync"

func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{inner: make(chan struct{}, capacity)}
}

type Semaphore struct {
	inner chan struct{}
}

func (s *Semaphore) Acquire() {
	s.inner <- struct{}{}
}

func (s *Semaphore) TryAcquire() bool {
	select {
	case s.inner <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Semaphore) Release() {
	select {
	case <-s.inner:
	default:
		panic("thread semaphore inconsistency: release before acquire!")
	}
}

type SemaphoreKey interface {
	Key() string
}

func NewSemaphorePool(semaCap int) *SemaphorePool {
	return &SemaphorePool{ss: make(map[string]*Semaphore), semaCap: semaCap}
}

type SemaphorePool struct {
	ss      map[string]*Semaphore
	semaCap int
	mu      sync.Mutex
}

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

func (p *SemaphorePool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// grab all semaphores and hold
	for _, s := range p.ss {
		s.Acquire()
	}
}
