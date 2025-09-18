package validator

import "sync"

var seqMapPool = sync.Pool{New: func() any { return make(map[int]string) }}

func GetSeqMap() map[int]string {
	return seqMapPool.Get().(map[int]string)
}

func PutSeqMap(m map[int]string) {
	for k := range m {
		delete(m, k)
	}
	seqMapPool.Put(m)
}
