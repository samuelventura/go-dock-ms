package main

import "sync"

type Result struct {
	err    error
	close  func()
	closed chan interface{}
}

type Closer struct {
	channel chan interface{}
	action  func()
	flag    bool
	mutex   sync.Mutex
}

func NewCloser(action func()) *Closer {
	c := &Closer{}
	c.action = action
	c.channel = make(chan interface{})
	return c
}

func (c *Closer) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.flag {
		c.flag = true
		close(c.channel)
		c.action()
	}
}

func (c *Closer) Result() *Result {
	result := &Result{}
	result.closed = c.channel
	result.close = c.Close
	return result
}

func (c *Closer) Channel() chan interface{} {
	return c.channel
}
