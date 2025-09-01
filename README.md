# nqueue 队列库详细说明

## 概述

`nqueue` 是一个基于 Go 语言实现的泛型队列库，采用链表结构存储元素，通过读写锁和条件变量保证并发安全性，支持阻塞和非阻塞两种出队模式，适用于各类并发场景下的任务调度和消息传递。

## 核心组件

### 1. 错误定义

```go
var (
    ErrQueueClosed      = errors.New("queue is closed")      // 队列已关闭错误
    ErrQueueClosedEmpty = errors.New("queue is closed and empty") // 队列已关闭且为空错误
)
```

### 2. 函数类型定义

```go
// DequeueFunc 用于处理出队元素的回调函数
// 参数: 出队元素、队列是否关闭
// 返回值: 是否继续处理下一个元素
type DequeueFunc[T any] func(T, bool) bool
```

### 3. 节点结构体

```go
// node 队列中的节点结构
type node[T any] struct {
    value T        // 节点存储的值
    next  *node[T] // 指向下一个节点的指针
}
```

### 4. 队列结构体

```go
// NQueue 泛型队列实现
type NQueue[T any] struct {
    head      *node[T]     // 队列头节点
    tail      *node[T]     // 队列尾节点
    status    bool         // 队列状态(true:打开, false:关闭)
    count     int64        // 元素数量
    recvLock  sync.RWMutex // 读写锁(并发安全控制)
    nodePool  sync.Pool    // 节点对象池(减少内存分配)
    zeroValue T            // 泛型零值
    recvCond  *sync.Cond   // 条件变量(用于阻塞等待)
}
```

## 核心方法实现

### 1. 队列创建

```go
// NewNQueue 创建新队列实例
func NewNQueue[T any]() *NQueue[T] {
    q := &NQueue[T]{}
    q.status = true                        // 初始状态为打开
    q.count = 0                            // 初始元素数量为0
    q.recvCond = sync.NewCond(&q.recvLock) // 绑定条件变量到读写锁
    q.nodePool = sync.Pool{
        New: func() any {
            return &node[T]{
                value: q.zeroValue,
                next:  nil,
            }
        },
    }
    return q
}
```

### 2. 入队操作

```go
// Enqueue 向队列尾部插入元素
func (q *NQueue[T]) Enqueue(v T) error {
    q.recvLock.Lock()
    defer q.recvLock.Unlock()

    if !q.status {
        return ErrQueueClosed // 队列关闭时返回错误
    }

    // 从对象池获取节点
    n := q.nodePool.Get().(*node[T])
    n.value = v
    n.next = nil

    // 更新队列头尾指针
    if q.head == nil {
        q.head = n
    } else {
        if q.tail == nil {
            q.tail = n
            q.head.next = q.tail
        } else {
            oldTail := q.tail
            oldTail.next = n
            q.tail = n
        }
    }

    q.count++
    q.recvCond.Broadcast() // 通知等待的goroutine
    return nil
}
```

### 3. 出队操作

#### 非阻塞出队

```go
// Dequeue 非阻塞出队
func (q *NQueue[T]) Dequeue() (t T, ok bool, isClose bool) {
    t, ok, isClose = q.dequeue()
    return
}
```

#### 阻塞出队

```go
// DequeueWait 阻塞出队，直到有元素或队列关闭
func (q *NQueue[T]) DequeueWait() (t T, ok bool, isClose bool) {
    for {
        t, ok, isClose = q.dequeue()
        if ok || isClose {
            return
        }

        q.recvLock.Lock()
        if q.status && q.count == 0 {
            q.recvCond.Wait() // 阻塞等待通知
        }
        q.recvLock.Unlock()
    }
}
```

#### 批量处理出队

```go
// DequeueFunc 批量处理出队元素
func (q *NQueue[T]) DequeueFunc(fn DequeueFunc[T]) (err error) {
    for {
        t, ok, isClose := q.dequeue()
        if ok {
            if !fn(t, isClose) {
                return
            }
        } else if isClose {
            return ErrQueueClosedEmpty
        }

        q.recvLock.Lock()
        if q.status && q.count == 0 {
            q.recvCond.Wait()
        }
        q.recvLock.Unlock()
    }
}
```

### 4. 队列关闭

```go
// Close 关闭队列并通知所有等待的goroutine
func (q *NQueue[T]) Close() {
    q.recvLock.Lock()
    defer q.recvLock.Unlock()
    q.status = false
    q.recvCond.Broadcast() // 广播通知所有等待者
}
```

## 并发安全机制

1. **读写锁 (`sync.RWMutex`)**: 保护队列的所有状态修改和读取操作
2. **条件变量 (`sync.Cond`)**: 实现阻塞出队时的等待-通知机制
3. **节点池 (`sync.Pool`)**: 复用节点对象，减少内存分配和GC开销
4. **状态标记**: 通过`status`字段控制队列生命周期，关闭后拒绝入队操作

## 性能优化点

- 采用链表结构，入队和出队操作均为O(1)时间复杂度
- 使用`sync.Pool`复用节点，减少内存分配次数
- 读写锁分离读写操作，提高并发性能
- 条件变量避免忙等，降低CPU消耗

## 使用场景

- 多生产者-多消费者模型
- 任务调度系统
- 异步消息处理
- 并发请求缓冲
- 限流控制