// core/sdns/forward_worker.go

package sdns

import (
	"sync"
)

// ForwardWorkerPool 专用的DNS转发协程池
type ForwardWorkerPool struct {
	taskChan    chan *DNSForwardTask
	workerCount int
	wg          sync.WaitGroup
	shutdown    bool
}

// NewForwardWorkerPool 创建新的DNS转发协程池
func NewForwardWorkerPool(workerCount int) *ForwardWorkerPool {
	if workerCount <= 0 {
		workerCount = 50000 // 默认值
	}

	pool := &ForwardWorkerPool{
		taskChan:    make(chan *DNSForwardTask, workerCount*2), // 队列大小为协程数的2倍
		workerCount: workerCount,
		shutdown:    false,
	}

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker 工作协程
func (p *ForwardWorkerPool) worker() {
	defer p.wg.Done()

	for task := range p.taskChan {
		task.Process()
	}
}

// SubmitTask 提交任务到协程池，处理协程池被占满的情况
func (p *ForwardWorkerPool) SubmitTask(task *DNSForwardTask) {
	select {
	case p.taskChan <- task:
		// 任务已加入队列
	default:
		// 队列已满，直接在当前协程中执行
		task.Process()
	}
}

// Close 关闭协程池
func (p *ForwardWorkerPool) Close() {
	p.shutdown = true
	close(p.taskChan)
	p.wg.Wait()
}
