package health

import "sync/atomic"

var overloaded int32
var latencyOverloaded int64

func SetOverloaded(v bool) {
	if v {
		atomic.StoreInt32(&overloaded, 1)
	} else {
		atomic.StoreInt32(&overloaded, 0)
	}
}

func SetLatencyOverloaded(v bool, latency int64) {
	if v {
		atomic.StoreInt32(&overloaded, 1)
		atomic.StoreInt64(&latencyOverloaded, latency)
	} else {
		atomic.StoreInt32(&overloaded, 0)
	}
}

func IsOverloaded(
	warningLatency int64,
	maxLatency int64,
) (int64, int64) {
	if atomic.LoadInt32(&overloaded) == 1 {
		return 2, 0
	}
	latency := atomic.LoadInt64(&latencyOverloaded)
	if latency >= maxLatency {
		return 2, latency // overload
	}
	if latency > warningLatency {
		return 1, latency // warning
	}
	return 0, 0
}
