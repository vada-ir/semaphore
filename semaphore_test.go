package semaphore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUsage(t *testing.T) {

	s := NewSemaphore(10)
	assert.Equal(t, 10, s.PermitCount())

	s.Acquire(9)

	var unlocked bool
	var fun = make(chan struct{})
	go func() {
		s.Acquire(10)
		assert.True(t, unlocked)
		fun <- struct{}{}
	}()

	unlocked = true
	s.Release(9)

	<-fun

	c := s.TryAcquire(10, time.Millisecond)
	assert.Equal(t, 0, c)

	go func() {
		c := s.TryAcquire(10, time.Second)
		assert.Equal(t, 5, c)
		fun <- struct{}{}
	}()

	c = s.TryRelease(5, time.Millisecond)
	assert.Equal(t, 5, c)
	<-fun

	c = s.TryRelease(15, time.Millisecond)
	assert.Equal(t, 10, c)

	c = s.TryAcquire(2, time.Second)
	assert.Equal(t, 2, c)

	s.TryRelease(2, time.Millisecond)
}

func TestResizable(t *testing.T) {
	r := NewResizableSemaphore(10)

	r.Acquire(10)

	//	var unlocked = false
	var fun = make(chan struct{})
	go func() {
		r.Acquire(3)
		fun <- struct{}{}
	}()

	r.Resize(14)
	<-fun

	assert.Equal(t, 14, r.PermitCount())
	assert.Equal(t, true, r.Stable())
	r.Resize(5)

	assert.Equal(t, false, r.Stable())
	r.Release(13)

	assert.Equal(t, true, r.Stable())
}
