// Package semaphore create a simple semaphore with channels.
package semaphore

import "time"

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
	TryRelease(int) int
	// Wait for all permits. this is a simple hack, and maybe removed in next changes.
	// for wait, its better to use sync.WaitGroup.
	Wait()
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
	c chan empty
}

type resizable struct {
	semaphore

	stable int
}

func (s *semaphore) PermitCount() int {
	return cap(s.c)
}

func (s *semaphore) Acquire(n int) {
	var write = func() (result bool) {
		defer func() {
			if e := recover(); e != nil {
				result = false
				return
			}

			result = true
		}()
		s.c <- empty{}
		return
	}
	for i := 0; i < n; {
		if write() {
			i++
		}
	}
}

func (s *semaphore) TryAcquire(n int, d time.Duration) int {
	var total int
	dc := time.After(d)

	var write = func() (result bool, finished bool) {
		defer func() {
			if e := recover(); e != nil {
				result = false
				return
			}

			result = true
		}()
		select {
		case s.c <- empty{}:
		case <-dc:
			finished = true
		}
		return
	}
	for i := 0; i < n; {
		result, finished := write()
		if finished {
			return total
		}
		if result {
			total++
			i++
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

func (s *semaphore) TryRelease(n int) int {
	var total int
	for i := 0; i < n; {
		select {
		case _, ok := <-s.c:
			if ok {
				total++
				i++
			}
		default:
			return total
		}
	}

	return total
}

func (s *semaphore) Wait() {
	all := s.PermitCount()
	s.Acquire(all)
	s.TryRelease(all)
}

func (s *resizable) Resize(n int) {
	s.stable++
	// First create new channel
	c := make(chan empty, n)
	// Swap channels
	c, s.c = s.c, c
	// Close the old one
	close(c)
	// its time to copy all permits from the old channel to new channel
	go func() {
		var count int
		for {
			// c is a closed channel, it return as soon as possible, no need to select here
			_, ok := <-c
			if !ok { // false means we reach the end of closed channel.
				break
			}
			count++
		}
		if count > 0 {
			s.Acquire(count)
		}
		s.stable--
	}()
}

func (s *resizable) Stable() bool {
	return s.stable == 0
}

// NewSemaphore return a new semaphore with requested size
func NewSemaphore(size int) Semaphore {
	return &semaphore{make(chan empty, size)}
}

// NewResizableSemaphore return a resizable semaphore with requested size
func NewResizableSemaphore(size int) ResizableSemaphore {
	return &resizable{semaphore{make(chan empty, size)}, 0}
}
