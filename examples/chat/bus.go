package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/ktsivkov/websocket_manager"
)

var (
	ErrUsernameTaken  = errors.New("username taken")
	ErrClientNotFound = errors.New("client not found")
)

type Bus struct {
	clients     *sync.Map
	wg          *sync.WaitGroup
	activeCount *atomic.Uint64
}

func NewBus() *Bus {
	return &Bus{
		clients:     &sync.Map{},
		wg:          &sync.WaitGroup{},
		activeCount: &atomic.Uint64{},
	}
}

func (b *Bus) Notify(target string, msg websocket_manager.Message) error {
	val, found := b.clients.Load(target)
	if !found {
		return fmt.Errorf("%w: %s", ErrClientNotFound, target)
	}
	val.(*Client).SendMessage(msg)
	return nil
}

func (b *Bus) NotifyAll(msg websocket_manager.Message, filtered ...string) {
	b.clients.Range(func(key, value interface{}) bool {
		if slices.Contains(filtered, key.(string)) {
			return true
		}

		value.(*Client).SendMessage(msg)
		return true
	})
}

func (b *Bus) Subscribe(client *Client) error {
	if _, taken := b.clients.LoadOrStore(client.Username(), client); taken {
		return fmt.Errorf("%w: %s", ErrUsernameTaken, client.Username())
	}

	b.wg.Add(1)
	b.activeCount.Add(1)
	return nil
}

func (b *Bus) Unsubscribe(client *Client) error {
	if _, ok := b.clients.LoadAndDelete(client.Username()); !ok {
		return fmt.Errorf("%w: %s", ErrClientNotFound, client.Username())
	}

	b.wg.Done()
	b.activeCount.Store(b.activeCount.Load() - 1)
	return nil
}

func (b *Bus) Contains(username string) bool {
	_, ok := b.clients.Load(username)
	return ok
}

func (b *Bus) Active(filtered ...string) []string {
	clients := make([]string, 0)
	b.clients.Range(func(key, value interface{}) bool {
		if val := key.(string); !slices.Contains(filtered, val) {
			clients = append(clients, val)
		}

		return true
	})

	return clients
}

func (b *Bus) Wait() {
	b.wg.Wait()
}

func (b *Bus) Shutdown(ctx context.Context) error {
	ch := make(chan struct{}, 1)
	go func() {
		b.clients.Range(func(key, value interface{}) bool {
			value.(*Client).NotifyShutdown()
			return true
		})
		b.wg.Wait()
		ch <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}

func (b *Bus) ActiveCount() uint64 {
	return b.activeCount.Load()
}
