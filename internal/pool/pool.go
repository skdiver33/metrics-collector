// Package pool - описывет структуру хранения тяжелых объетов
// ограниченных интерфейсом Reset()
package pool

import "slices"

type PoolStruct[T interface{ Reset() }] struct {
	store []*T
}

func (pl *PoolStruct[T]) New() *PoolStruct[T] {
	return &PoolStruct[T]{store: make([]*T, 0)}
}

func (pl *PoolStruct[T]) Get() *T {
	size := len(pl.store)
	if size > 0 {
		retVal := pl.store[size-1]
		pl.store = slices.Delete(pl.store, size-1, size)
		(*retVal).Reset()
		return retVal
	}
	return nil
}

func (pl *PoolStruct[T]) Put(newVal *T) {
	pl.store = append(pl.store, newVal)
}
