package timedmap

import (
	"time"
)

type Ticker interface {
	Chan() <-chan time.Time
	Stop()
}

type DefaultTicker time.Ticker

func (t *DefaultTicker) Chan() <-chan time.Time {
	return (*time.Ticker)(t).C
}
func (t *DefaultTicker) Stop() {
	(*time.Ticker)(t).Stop()
}

type Cleaner struct {
	ticker   Ticker
	stopChan chan struct{}
	onTickQ  []func()
}

type Cleanable interface {
	Cleanup()
}

func NewCleaner(dura time.Duration) *Cleaner {
	return NewCleanerCustom((*DefaultTicker)(time.NewTicker(dura)))
}

func NewCleanerCustom(ticker Ticker) *Cleaner {
	return &Cleaner{
		ticker:   ticker,
		stopChan: make(chan struct{}),
	}
}

func (c *Cleaner) AddCallback(onTicks ...func()) {
	c.onTickQ = append(c.onTickQ, onTicks...)
}
func (c *Cleaner) AddCleanable(cleanable Cleanable) {
	c.AddCallback(cleanable.Cleanup)
}

func (c *Cleaner) Start() {
	go func() {
		for {
			select {
			case <-c.ticker.Chan():
				for _, fn := range c.onTickQ {
					fn()
				}
			case <-c.stopChan:
				break
			}
		}
	}()
}

func (c *Cleaner) Stop() {
	c.ticker.Stop()
	close(c.stopChan)
}
