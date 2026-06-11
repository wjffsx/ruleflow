package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
)

// shutdown 优雅关闭管理器（引擎内部实现）
type shutdown struct {
	stateVal  atomic.Int32
	wg        sync.WaitGroup
	once      sync.Once
	err       error
	errMu     sync.Mutex
	startAt   time.Time
}

// newShutdown 创建关闭管理器
func newShutdown() *shutdown {
	return &shutdown{startAt: time.Now()}
}

// markShutdown 标记为关闭中。返回 true 表示首次调用。
func (s *shutdown) markShutdown() bool {
	first := false
	s.once.Do(func() {
		first = true
		s.stateVal.Store(int32(contract.ShutdownStateShuttingDown))
	})
	return first
}

// begin 标记一个评估开始。关闭后返回 false。
func (s *shutdown) begin() bool {
	if s.stateVal.Load() != int32(contract.ShutdownStateRunning) {
		return false
	}
	s.wg.Add(1)
	return true
}

// end 标记一个评估结束
func (s *shutdown) end() {
	s.wg.Done()
}

// isShuttingDown 检查是否正在关闭或已关闭
func (s *shutdown) isShuttingDown() bool {
	state := s.stateVal.Load()
	return state == int32(contract.ShutdownStateShuttingDown) || state == int32(contract.ShutdownStateShutdown)
}

// isShutdown 检查是否已完全关闭
func (s *shutdown) isShutdown() bool {
	return s.stateVal.Load() == int32(contract.ShutdownStateShutdown)
}

// getState 返回当前状态
func (s *shutdown) getState() contract.ShutdownState {
	return contract.ShutdownState(s.stateVal.Load())
}

// wait 等待所有进行中的评估完成或上下文取消。
func (s *shutdown) wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.stateVal.Store(int32(contract.ShutdownStateShutdown))
		return nil
	case <-ctx.Done():
		s.errMu.Lock()
		s.err = ctx.Err()
		s.errMu.Unlock()
		s.stateVal.Store(int32(contract.ShutdownStateShutdown))
		return ctx.Err()
	}
}

// lastError 返回最近一次错误
func (s *shutdown) lastError() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}
