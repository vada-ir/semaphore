# Semaphore

A simple semaphore implementation in golang using channels

[![Build 
Status](https://travis-ci.org/vada-ir/semaphore.svg?branch=master)](https://travis-ci.org/vada-ir/semaphore)
[![Coverage 
Status](https://coveralls.io/repos/vada-ir/semaphore/badge.svg?branch=master)](https://coveralls.io/r/vada-ir/semaphore?branch=master)

## Usage 

```go 

package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/fzerorubigd/semaphore"
)

func crawl(index int, s semaphore.Semaphore) {
	defer s.Release(1)
	fmt.Printf("crawling the site #%d\n", index)
	r := time.Duration(rand.Intn(2))
	time.Sleep(time.Second * r)
	fmt.Printf("job #%d is done\n", index)
}

func main() {
	rand.Seed(time.Now().Unix())
	s := semaphore.NewSemaphore(5)
	for i := 1; i < 100; i++ {
		s.Acquire(1)

		go crawl(i, s)
	}

	s.Wait()
}

```
