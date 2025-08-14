package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

const HI_VAL = 50000_0000

func repeatFn[T any](done <-chan bool, aFn func() T) <-chan T {
	repeatStream := make(chan T)

	go func() {
		defer close(repeatStream)
		for {
			select {
			case <-done:
				return
			case repeatStream <- aFn():
			}
		}
	}()
	return repeatStream
}

func take[T any](done <-chan bool, inputChan <-chan T, numValuesToTake int) <-chan T {
	takenChan := make(chan T)
	go func() {
		defer close(takenChan)
		for i := numValuesToTake; i > 0; i-- {
			select {
			case <-done:
				return
			case takenChan <- <-inputChan:
			}
		}
	}()

	return takenChan
}

func primeFinder[T int](done <-chan bool, randomInts <-chan int) <-chan int {

	primePredicateFn := func(aNumber int) bool {
		for i := aNumber - 1; i > 1; i-- {
			if aNumber%i == 0 {
				return false
			}
		}
		return true
	}
	primesChan := make(chan int)
	go func() {
		defer close(primesChan)
		for {
			select {
			case <-done:
				return
			case val := <-randomInts:
				if primePredicateFn(val) {
					primesChan <- val
				}
			}
		}
	}()
	return primesChan
}

func fanIn[T int](done <-chan bool, input ...<-chan T) <-chan T {
	fannedInChan := make(chan T)

	var wg sync.WaitGroup
	wg.Add(len(input))

	drainChan := func(aChan <-chan T) {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case fannedInChan <- <-aChan:
			}
		}
	}

	for _, ch := range input {
		go drainChan(ch)
	}

	go func() {
		wg.Wait()
		close(fannedInChan)
	}()

	return fannedInChan
}

func pipeLine() {
	generatorFn := func() int {
		return rand.Intn(HI_VAL)
	}

	doneChan := make(chan bool)

	randomStream := repeatFn(doneChan, generatorFn)
	primesStream := primeFinder[int](doneChan, randomStream)

	for v := range take(doneChan, primesStream, 10) {
		fmt.Println("Prime Generated ", v)
	}

	close(doneChan)
}

func fanInAndOut() {
	generatorFn := func() int {
		return rand.Intn(HI_VAL)
	}

	doneChan := make(chan bool)

	randomStream := repeatFn(doneChan, generatorFn)

	numCpuCores := runtime.NumCPU()
	fmt.Println("Num CPU ", numCpuCores)

	fanOutChans := make([]<-chan int, numCpuCores)

	for i := range numCpuCores {
		fanOutChans[i] = primeFinder[int](doneChan, randomStream)
	}

	for v := range take(doneChan, fanIn(doneChan, fanOutChans...), 10) {
		fmt.Println("Prime Generated ", v)
	}

	close(doneChan)
}

func main() {

	startTime := time.Now()
	defer func() {
		fmt.Println(time.Since(startTime))
	}()

	//pipeLine()

	fanInAndOut()
	time.Sleep(5 * time.Second)
	fmt.Println("Num Goroutines ", runtime.NumGoroutine())
}
