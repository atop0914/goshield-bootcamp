package metrics

import (
	"sync/atomic"
)

func atomicAddUint64(addr *uint64, delta uint64) {
	atomic.AddUint64(addr, delta)
}

func atomicLoadUint64(addr *uint64) uint64 {
	return atomic.LoadUint64(addr)
}

func atomicAddInt64(addr *int64, delta int64) {
	atomic.AddInt64(addr, delta)
}

func atomicLoadInt64(addr *int64) int64 {
	return atomic.LoadInt64(addr)
}

func atomicStoreInt64(addr *int64, val int64) {
	atomic.StoreInt64(addr, val)
}
