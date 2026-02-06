// core/sdns/workerpool.go

package sdns

import (
	"sync"
	"time"

	"github.com/miekg/dns"
)

// workerPool 对象池用于复用DNSWorker对象
var workerPool = sync.Pool{
	New: func() interface{} {
		return &DNSWorker{}
	},
}

// DNSWorker DNS工作协程
type DNSWorker struct {
	handler   dns.Handler
	w         dns.ResponseWriter
	r         *dns.Msg
	clientIP  string
	startTime time.Time
}

// Process 处理DNS请求
func (w *DNSWorker) Process() {
	// 尝试将客户端IP地址传递给DNSHandler
	if handlerWithClientIP, ok := w.handler.(interface{ SetClientIP(string) }); ok {
		handlerWithClientIP.SetClientIP(w.clientIP)
	}

	w.handler.ServeDNS(w.w, w.r)
}

// WorkerPool 协程池
type WorkerPool struct {
	taskChan        chan *DNSWorker
	workerCount     int
	queueMultiplier int
	wg              sync.WaitGroup
	shutdown        bool
	stmu            sync.Mutex
	statsMu         sync.Mutex
	mu              sync.RWMutex
	stats           *PoolStats
}

// PoolStats 协程池统计信息
type PoolStats struct {
	TotalTasks     int64         `json:"totalTasks"`
	CompletedTasks int64         `json:"completedTasks"`
	FailedTasks    int64         `json:"failedTasks"`
	AverageLatency time.Duration `json:"averageLatency"`
	QueueLength    int           `json:"queueLength"`
	ActiveWorkers  int           `json:"activeWorkers"`
	StartTime      time.Time     `json:"startTime"`
	LastTaskTime   time.Time     `json:"lastTaskTime"`
}

// NewWorkerPool 创建新的协程池
func NewWorkerPool(workerCount, queueMultiplier int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1000 // 默认值
	}
	if queueMultiplier <= 0 {
		queueMultiplier = 2 // 默认值
	}

	// 队列大小为当前协程数的 queueMultiplier 倍
	queueSize := workerCount * queueMultiplier

	pool := &WorkerPool{
		taskChan:        make(chan *DNSWorker, queueSize),
		workerCount:     workerCount,
		queueMultiplier: queueMultiplier,
		shutdown:        false,
		stats: &PoolStats{
			StartTime: time.Now(),
		},
	}

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker 工作协程
func (p *WorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case task, ok := <-p.taskChan:
			if !ok {
				return
			}

			// 处理任务
			p.processTask(task)
		}
	}
}

// processTask 处理单个任务
func (p *WorkerPool) processTask(task *DNSWorker) {
	p.mu.Lock()
	p.stats.ActiveWorkers++
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.stats.ActiveWorkers--
		p.stats.CompletedTasks++
		p.stats.LastTaskTime = time.Now()
		if !task.startTime.IsZero() {
			// 计算任务处理时间
			latency := time.Since(task.startTime)
			// 更新平均延迟（简单移动平均）
			if p.stats.CompletedTasks > 1 {
				p.stats.AverageLatency = (p.stats.AverageLatency*time.Duration(p.stats.CompletedTasks-1) + latency) / time.Duration(p.stats.CompletedTasks)
			} else {
				p.stats.AverageLatency = latency
			}
		}
		p.mu.Unlock()

		// 重置并回收DNSWorker对象到对象池
		if task != nil {
			task.handler = nil
			task.w = nil
			task.r = nil
			task.clientIP = ""
			task.startTime = time.Time{}
			workerPool.Put(task)
		}
	}()

	// 处理任务
	if task != nil {
		// 设置任务超时控制（5秒）
		done := make(chan struct{})
		go func() {
			// 检查是否是Task接口的实现
			if taskHandler, ok := task.handler.(Task); ok {
				taskHandler.Process()
			} else {
				task.Process()
			}
			close(done)
		}()

		// 等待任务完成或超时
		select {
		case <-done:
			// 任务正常完成
		case <-time.After(5 * time.Second):
			// 任务超时
			p.mu.Lock()
			p.stats.FailedTasks++
			p.mu.Unlock()

			// 返回服务器失败错误
			if task.w != nil && task.r != nil {
				m := new(dns.Msg)
				m.SetRcode(task.r, dns.RcodeServerFailure)
				task.w.WriteMsg(m)
			}
		}
	}
}

// Submit 提交DNS请求
func (p *WorkerPool) Submit(handler dns.Handler, w dns.ResponseWriter, r *dns.Msg) {
	worker := workerPool.Get().(*DNSWorker)
	worker.handler = handler
	worker.w = w
	worker.r = r
	worker.clientIP = "unknown"
	worker.startTime = time.Now()

	p.submitTask(worker)
}

// SubmitWithClientIP 提交带客户端IP地址的DNS请求
func (p *WorkerPool) SubmitWithClientIP(handler dns.Handler, w dns.ResponseWriter, r *dns.Msg, clientIP string) {
	worker := workerPool.Get().(*DNSWorker)
	worker.handler = handler
	worker.w = w
	worker.r = r
	worker.clientIP = clientIP
	worker.startTime = time.Now()

	p.submitTask(worker)
}

// submitTask 提交任务到队列
func (p *WorkerPool) submitTask(task *DNSWorker) {
	p.mu.Lock()
	p.stats.TotalTasks++
	p.mu.Unlock()

	// 带超时的提交
	select {
	case p.taskChan <- task:
		// 请求已加入队列
	default:
		// 队列已满，尝试处理
		p.handleQueueFull(task)
	}
}

// handleQueueFull 处理队列满的情况
func (p *WorkerPool) handleQueueFull(task *DNSWorker) {
	// 队列已满，直接处理或返回错误
	if task.w != nil && task.r != nil {
		// 返回服务器失败错误
		m := new(dns.Msg)
		m.SetRcode(task.r, dns.RcodeServerFailure)
		task.w.WriteMsg(m)

		p.mu.Lock()
		p.stats.FailedTasks++
		p.mu.Unlock()
	}

	// 重置并回收DNSWorker对象到对象池
	if task != nil {
		task.handler = nil
		task.w = nil
		task.r = nil
		task.clientIP = ""
		task.startTime = time.Time{}
		workerPool.Put(task)
	}
}

// Task 任务接口
type Task interface {
	Process()
}

// SubmitTask 提交通用任务
func (p *WorkerPool) SubmitTask(task Task) {
	// 从对象池获取DNSWorker对象
	worker := workerPool.Get().(*DNSWorker)
	worker.handler = taskAdapter{task: task}
	worker.w = nil
	worker.r = nil
	worker.clientIP = "unknown"
	worker.startTime = time.Now()

	p.submitTask(worker)
}

// taskAdapter 任务适配器，将Task接口包装为dns.Handler接口
type taskAdapter struct {
	task Task
}

// ServeDNS 实现dns.Handler接口
func (a taskAdapter) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	a.task.Process()
}

// Close 关闭协程池
func (p *WorkerPool) Close() {
	if !p.shutdown {
		close(p.taskChan)
	}
	p.stmu.Lock()
	p.shutdown = true
	p.stmu.Unlock()

	p.wg.Wait()
}

// GetWorkerCount 获取当前工作协程数
func (p *WorkerPool) GetWorkerCount() int {
	return p.workerCount
}

// GetQueueLength 获取当前队列长度
func (p *WorkerPool) GetQueueLength() int {
	return len(p.taskChan)
}

// GetStats 获取协程池统计信息
func (p *WorkerPool) GetStats() *PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 创建统计信息的副本
	stats := *p.stats
	stats.QueueLength = len(p.taskChan)
	return &stats
}
