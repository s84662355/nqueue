package nqueue

type DequeueFunc[T any] func(t T, isClose bool) bool

type Queue[T any] interface {
	Close()
	Enqueue(T) error
	Dequeue() (t T, ok bool, isClose bool)
	DequeueWait() (t T, ok bool, isClose bool)
	DequeueFunc(fn DequeueFunc[T]) (err error)
	Count() int64
	Status() bool
}
