package cache4service

import (
	"context"
	"sync"
	"time"
)

type CacheForService[K ~string] interface {
	SetCustomKeeper(f func(ctx context.Context, key K, value any, expiration time.Duration) error)
	SetCustomRemover(f func(ctx context.Context, key K) error)
	ToSave(key K, value any, expiration time.Duration)
	ToDiscard(keys ...K)
	Watch()
	Wait()
}

type cacheService[K ~string] struct {
	wg            *sync.WaitGroup
	toDiscard     chan K
	toSave        chan valueToSave[K]
	customKeeper  func(ctx context.Context, key K, value any, expiration time.Duration) error
	customRemover func(ctx context.Context, key K) error
}

type valueToSave[K ~string] struct {
	key   K
	value any
	exp   time.Duration
}

func NewCacheForService[K ~string](
	customKeeper func(ctx context.Context, key K, value any, expiration time.Duration) error,
	customRemover func(ctx context.Context, key K) error,
	chanSize uint,
) CacheForService[K] {
	return &cacheService[K]{
		wg:        &sync.WaitGroup{},
		toDiscard: make(chan K, chanSize),
	}
}

func (cs *cacheService[K]) ToDiscard(keys ...K) {
	for _, key := range keys {
		cs.toDiscard <- key
	}
}

func (cs *cacheService[K]) Watch() {
	cs.wg.Add(2)

	go cs.watchKeeperChan()
	go cs.watchDiscardChan()
}

func (cs *cacheService[K]) Wait() {
	close(cs.toDiscard)
	close(cs.toSave)

	cs.wg.Wait()
}

func (cs *cacheService[K]) watchDiscardChan() {
	defer cs.wg.Done()

	for key := range cs.toDiscard {
		cs.customRemover(
			context.Background(),
			key,
		)
	}
}

func (cs *cacheService[K]) watchKeeperChan() {
	defer cs.wg.Done()

	for item := range cs.toSave {
		cs.customKeeper(
			context.Background(),
			item.key,
			item.value,
			item.exp,
		)
	}
}

func (cs *cacheService[K]) ToSave(key K, value any, expiration time.Duration) {
	cs.toSave <- valueToSave[K]{
		key:   key,
		value: value,
		exp:   expiration,
	}
}

func (cs *cacheService[K]) SetCustomKeeper(f func(ctx context.Context, key K, value any, expiration time.Duration) error) {
	cs.customKeeper = f
}

func (cs *cacheService[K]) SetCustomRemover(f func(ctx context.Context, key K) error) {
	cs.customRemover = f
}
