package main

import (
	"sync"
)

type ConnectionPool struct {
	mu   sync.RWMutex
	pool map[string]chan []byte
}

func (p *ConnectionPool) Has(key string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.pool[key]
	return ok
}

func (p *ConnectionPool) Register(key string) <-chan []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pool == nil {
		p.pool = make(map[string]chan []byte)
	}

	if ch, ok := p.pool[key]; ok {
		return ch
	}

	ch := make(chan []byte)
	p.pool[key] = ch
	return ch
}

func (p *ConnectionPool) Unregister(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ch, ok := p.pool[key]; ok {
		close(ch)
		delete(p.pool, key)
	}
}

func (p *ConnectionPool) UnregisterAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	wg := sync.WaitGroup{}
	for key := range p.pool {
		wg.Go(func() {
			p.Unregister(key)
		})
	}
	wg.Wait()
}

func (p *ConnectionPool) SendTo(key string, payload []byte) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if ch, ok := p.pool[key]; ok {
		ch <- payload
		return
	}
}

func (p *ConnectionPool) SendToAll(payload []byte) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	wg := sync.WaitGroup{}
	for _, ch := range p.pool {
		wg.Go(func() {
			ch <- payload
		})
	}
	wg.Wait()
}

func (p *ConnectionPool) Channel(key string) (<-chan []byte, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ch, ok := p.pool[key]
	return ch, ok
}
