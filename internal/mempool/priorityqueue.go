package mempool

import (
	"container/heap"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
)

// txHeap implements heap.Interface as a max-heap ordered by GasPrice (ties by Timestamp ascending).
type txHeap []*core.Transaction

func (h txHeap) Len() int { return len(h) }

func (h txHeap) Less(i, j int) bool {
	if h[i].GasPrice != h[j].GasPrice {
		return h[i].GasPrice > h[j].GasPrice // higher gas price first
	}
	return h[i].Timestamp < h[j].Timestamp // earlier timestamp first
}

func (h txHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *txHeap) Push(x interface{}) {
	*h = append(*h, x.(*core.Transaction))
}

func (h *txHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return x
}

// ensure heap.Interface is satisfied at compile time.
var _ heap.Interface = (*txHeap)(nil)
