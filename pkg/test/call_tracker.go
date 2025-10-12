package test

import "sync"

// CallTracker tracks method invocations
type CallTracker struct {
	mu    sync.RWMutex
	calls map[string]int
}

func NewCallTracker() *CallTracker {
	return &CallTracker{calls: make(map[string]int)}
}

func (ct *CallTracker) Record(method string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.calls[method]++
}

func (ct *CallTracker) Called(method string) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.calls[method]
}

func (ct *CallTracker) CalledOnce(method string) bool {
	return ct.Called(method) == 1
}

func (ct *CallTracker) CalledAtLeast(method string, n int) bool {
	return ct.Called(method) >= n
}

func (ct *CallTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.calls = make(map[string]int)
}
