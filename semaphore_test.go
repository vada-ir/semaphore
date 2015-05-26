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

	c = s.TryRelease(5)
	assert.Equal(t, 5, c)
	<-fun

	c = s.TryRelease(15)
	assert.Equal(t, 10, c)

	c = s.TryAcquire(2, time.Second)
	assert.Equal(t, 2, c)

	go func() {
		s.Wait()
		fun <- struct{}{}
	}()

	s.TryRelease(2)

	<-fun
}
