package common

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCreateUnavailable = errors.New("create func return nil")
)

type EventType int

const (
	BreakerAdded EventType = iota
	CacheAdded   EventType = iota
)

type NamedBreaker struct {
	Key     string
	Breaker Breaker
}
type NamedCache struct {
	Key   string
	Cache Cache
}

type (
	Breaker interface {
		Allow() error
		MarkSuccess()
		MarkFailed()
	}
	Cache interface {
		Get(key interface{}) interface{}
		Set(key interface{}, value interface{}, duration time.Duration)
	}
	Group struct {
		mu             sync.RWMutex
		breakerMap     map[string]Breaker
		breakerTmpl    BreakerFunc
		cacheMap       map[string]Cache
		cacheTmpl      CacheFunc
		eventReceivers []chan GroupEvent
	}

	GroupEvent struct {
		Event  EventType
		Member interface{}
	}

	BreakerFunc func(key string) Breaker
	CacheFunc   func(key string) Cache
)

func NewGroup() *Group {
	return &Group{
		breakerMap: make(map[string]Breaker),
		cacheMap:   make(map[string]Cache),
	}
}

func (g *Group) WithBreaker(breakerFunc BreakerFunc) *Group {
	g.breakerTmpl = breakerFunc
	return g
}

func (g *Group) WithCache(cacheFunc CacheFunc) *Group {
	g.cacheTmpl = cacheFunc
	return g
}

func (g *Group) GetOrNewBreaker(key string) (Breaker, error) {
	g.mu.RLock()
	breaker, ok := g.breakerMap[key]
	g.mu.RUnlock()
	if ok {
		return breaker, nil
	}
	g.mu.Lock()
	breaker = g.breakerTmpl(key)
	if breaker == nil {
		return nil, ErrCreateUnavailable
	}
	if _, ok = g.breakerMap[key]; !ok {
		g.breakerMap[key] = breaker
		go func() {
			g.sendEvent(BreakerAdded, breaker)
		}()
	}
	g.mu.Unlock()
	return breaker, nil
}

func (g *Group) GetOrNewCache(key string) (Cache, error) {
	g.mu.RLock()
	cache, ok := g.cacheMap[key]
	g.mu.RUnlock()
	if ok {
		return cache, nil
	}
	g.mu.Lock()
	cache = g.cacheTmpl(key)
	if cache == nil {
		return nil, ErrCreateUnavailable
	}
	if _, ok = g.cacheMap[key]; !ok {
		g.cacheMap[key] = cache
		go func() {
			g.sendEvent(CacheAdded, cache)
		}()
	}
	g.mu.Unlock()
	return cache, nil
}

func (g *Group) sendEvent(event EventType, member interface{}) {
	for _, receiver := range g.eventReceivers {
		receiver <- GroupEvent{event, member}
	}
}

func (g *Group) Subscribe() <-chan GroupEvent {
	eventReader := make(chan GroupEvent)
	output := make(chan GroupEvent, 100)

	go func() {
		for v := range eventReader {
			select {
			case output <- v:
			default:
				<-output
				output <- v
			}
		}
	}()
	g.eventReceivers = append(g.eventReceivers, eventReader)
	return output
}
