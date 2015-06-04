// Package semaphore create a simple semaphore with channels.
package semaphore

import (
	"sync"
	"time"
)

// Semaphore is for managing permit
type Semaphore interface {
	// PermitCount return the number of permit in this semaphore
	PermitCount() int
	//Acquire try to acquire n permit, block until all permits are available
	Acquire(int)
	// TryAcquire try to acquire n permit in a time limit. return the number of
	// permit it get in duration
	TryAcquire(int, time.Duration) int
	// Release release the permits. may called from anothe routine, so maybe it
	// blocked if the number of released permit is not available for release
	Release(int)
	// TryRelease try to release permits and if no permit already taken, return
	// the number of actually relaesed permits
	TryRelease(int, time.Duration) int
}

// ResizableSemaphore is a semaphore with resize ability, but it cost unstability after resize.
type ResizableSemaphore interface {
	Semaphore
	// Resize try to resize the semaphore to new size, if the acquired permits are more than
	// the current size, then it try to get the permits again. ITS NOT AN ACTUAL SEMAPHOR BEHAVIOR
	// in resize, there is no guarantee thet the permits are less than the PermitCount()
	// until it reach its stable phase
	Resize(int)
	// Stable means is this semaphore is in stable phase?
	Stable() bool
}

type empty struct{}

type semaphore struct {
	// the main chan wit size
	c chan empty
	// a zero size channel, its the entry point of semaphore
	e chan empty
	// A lock, used at resize semaphore
	sync.RWMutex
}

type resizable struct {
	semaphore

	stable int
}

func (s *semaphore) PermitCount() int {
	s.RLock()
	defer s.RUnlock()

	return cap(s.c) + 1
}

func (s *semaphore) Acquire(n int) {
	for i := 0; i < n; i++ {
		s.e <- empty{}
	}
}

func (s *semaphore) TryAcquire(n int, d time.Duration) int {
	var total int
	dc := time.After(d)
	for i := 0; i < n; i++ {
		select {
		case s.e <- empty{}:
			total++
		case <-dc:
			return total
		}
	}
	return total
}

func (s *semaphore) Release(n int) {
	for i := 0; i < n; {
		_, ok := <-s.c
		if ok {
			i++
		}
	}
}

func (s *semaphore) TryRelease(n int, d time.Duration) int {
	var total int
	dc := time.After(d)
	for i := 0; i < n; {
		select {
		case _, ok := <-s.c:
			if ok {
				total++
				i++
			}
		case <-dc:
			return total
		}
	}

	return total
}

func (s *semaphore) loop() {
	for {
		// Read from the entry channel
		tmp := <-s.e

		s.RLock()
		s.c <- tmp
		s.RUnlock()
	}
}

func (s *resizable) Resize(n int) {
	s.Lock()
	defer s.Unlock()

	s.stable++
	// close the sized channel
	close(s.c)
	// its time to copy all permits from the old channel to new channel
	var count int
	for {
		// c is a closed channel, it return as soon as possible, no need to select here
		_, ok := <-s.c
		if !ok { // false means we reach the end of closed channel.
			break
		}
		count++
	}
	c := make(chan empty, n-1)

	var remain int
	for i := 0; i < count; i++ {
		select {
		case c <- empty{}:
		default:
			remain++
		}
	}
	s.c = c
	// try to acquire other value
	if remain > 0 {
		// TODO stable
		go func() {
			s.Acquire(remain)
			// ok its stable now!
			s.Lock()
			defer s.Unlock()
			s.stable--
		}()
	} else {
		// its already locked
		s.stable--
	}
}

func (s *resizable) Stable() bool {
	s.RLock()
	defer s.RUnlock()

	return s.stable == 0
}

// NewSemaphore return a new semaphore with requested size
func NewSemaphore(size int) Semaphore {
	s := semaphore{make(chan empty, size-1), make(chan empty), sync.RWMutex{}}
	go s.loop()
	return &s
}

// NewResizableSemaphore return a resizable semaphore with requested size
func NewResizableSemaphore(size int) ResizableSemaphore {
	s := resizable{semaphore{make(chan empty, size-1), make(chan empty), sync.RWMutex{}}, 0}
	go s.loop()
	return &s
}
