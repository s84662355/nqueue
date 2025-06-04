package Queue

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// / go test -run TestNqueue -v
func TestNqueue(t *testing.T) {
	qqqq := NewNQueue[int]()
	iiii := 0
	done := make(chan struct{})
	go func() {
		defer close(done)
		qqqq.DequeueFunc(func(fff int, isClose bool) bool {
			iiii += fff
			return true
		})
	}()

	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 1; i < 100; i++ {
				qqqq.Enqueue(i)
			}
		}()

	}

	wg.Wait()
	qqqq.Close()
	<-done
	fmt.Println(iiii)
}

// go test -run TestLockFreeQueue -v
func TestLockFreeQueue(t *testing.T) {
	// 创建一个存储整数的无锁队列
	var queue Queue[int] = nil

	queue = NewNQueue[int]()
	var wg, wg1 sync.WaitGroup

	var count atomic.Int64

	// 并发入队操作
	numEnqueuers := 100000
	wg.Add(numEnqueuers)
	go func() {
		for i := 1; i <= numEnqueuers; i++ {
			go func(id int) {
				defer wg.Done()

				// for i := 0; i < 5; i++ {
				queue.Enqueue(id)
				//}
			}(i)
		}

		fmt.Println(time.Now())
	}()

	// 并发出队操作
	numDequeuers := 1
	wg1.Add(numDequeuers)
	go func() {
		for i := 0; i < numDequeuers; i++ {
			go func(id int) {
				defer wg1.Done()
				for {
					if value, ok, _ := queue.DequeueWait(); ok {
						// fmt.Printf("Dequeuer %d removed %d from the queue\n", id, value)
						count.Add(int64(value))
					} else {
						return
					}
				}
			}(i)
		}
	}()

	// 等待所有 goroutine 完成
	wg.Wait()
	queue.Close()
	wg1.Wait()
	fmt.Println(queue.Count(), count.Load())
	fmt.Println(time.Now())
}

// go test -run TestLockFreeQueueManay -v
func TestLockFreeQueueManay(t *testing.T) {
	const shareSize = 32
	queueArr := [shareSize]Queue[int]{}
	for i := 0; i < shareSize; i++ {
		queueArr[i] = NewNQueue[int]()
	}

	var wg, wg1 sync.WaitGroup

	var count atomic.Int64
	var qc atomic.Uint64

	goChan := make(chan struct{})

	// 并发入队操作
	numEnqueuers := 500000
	wg.Add(numEnqueuers)
	go func() {
		for i := 1; i <= numEnqueuers; i++ {
			go func(id int) {
				defer wg.Done()
				<-goChan

				for i := 0; i < 1000; i++ {
					//	 time.Sleep(time.Duration((time.Now().UnixNano()%3)) * time.Second)
					queueArr[(qc.Add(1) % shareSize)].Enqueue(1)
				}
			}(i)
		}

		close(goChan)
		fmt.Println(time.Now())
	}()

	// 并发出队操作
	wg1.Add(shareSize)
	go func() {
		for i := 0; i < shareSize; i++ {
			go func(id int) {
				defer wg1.Done()
				for {
					if value, ok, isClose := queueArr[i].DequeueWait(); ok {
						// fmt.Printf("Dequeuer %d removed %d from the queue\n", id, value)
						count.Add(int64(value))
					} else if isClose {
						return
					}
				}
			}(i)
		}
	}()

	// 等待所有 goroutine 完成
	wg.Wait()
	for i := 0; i < shareSize; i++ {
		queueArr[i].Close()
	}

	wg1.Wait()
	fmt.Println(count.Load())
	fmt.Println(time.Now())
}
