// Package semaphore create a simple semaphore with channels.
package semaphore

import (
	"sync"
	"time"
)

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

type resizable struct {
	// the main chan wit size
	c chan empty
	// a zero size channel, its the entry point of semaphore
	e chan empty
	// A lock, used at resize semaphore
	sync.RWMutex

	stable int
}

func (r *resizable) PermitCount() int {
	r.RLock()
	defer r.RUnlock()

	return cap(r.c) + 1
}

func (r *resizable) Acquire(n int) {
	for i := 0; i < n; i++ {
		r.e <- empty{}
	}
}

func (r *resizable) TryAcquire(n int, d time.Duration) int {
	var total int
	dc := time.After(d)
	for i := 0; i < n; i++ {
		select {
		case r.e <- empty{}:
			total++
		case <-dc:
			return total
		}
	}
	return total
}

func (r *resizable) Release(n int) {
	for i := 0; i < n; {
		_, ok := <-r.c
		if ok {
			i++
		}
	}
}

func (r *resizable) TryRelease(n int, d time.Duration) int {
	var total int
	dc := time.After(d)
	for i := 0; i < n; {
		select {
		case _, ok := <-r.c:
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

func (r *resizable) loop() {
	for {
		// Read from the entry channel
		tmp := <-r.e

		r.RLock()
		r.c <- tmp
		r.RUnlock()
	}
}

func (r *resizable) Resize(n int) {
	r.Lock()
	defer r.Unlock()

	r.stable++
	// close the sized channel
	close(r.c)
	// its time to copy all permits from the old channel to new channel
	var count int
	for {
		// c is a closed channel, it return as soon as possible, no need to select here
		_, ok := <-r.c
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
	r.c = c
	// try to acquire other value
	if remain > 0 {
		// TODO stable
		go func() {
			r.Acquire(remain)
			// ok its stable now!
			r.Lock()
			defer r.Unlock()
			r.stable--
		}()
	} else {
		// its already locked
		r.stable--
	}
}

func (r *resizable) Stable() bool {
	r.RLock()
	defer r.RUnlock()

	return r.stable == 0
}

// NewResizableSemaphore return a resizable semaphore with requested size
func NewResizableSemaphore(size int) ResizableSemaphore {
	r := resizable{make(chan empty, size-1), make(chan empty), sync.RWMutex{}, 0}
	go r.loop()
	return &r
}
