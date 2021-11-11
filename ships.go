package main

import (
	"sync"

	"github.com/samuelventura/go-tree"
)

type shipsDso struct {
	mutex *sync.Mutex
	ships map[string]tree.Node
}

type Ships interface {
	Del(name string, node tree.Node)
	Add(name string, node tree.Node)
	Count() int
}

func NewShips() Ships {
	dso := &shipsDso{}
	dso.mutex = &sync.Mutex{}
	dso.ships = make(map[string]tree.Node)
	return dso
}

func (dso *shipsDso) Add(name string, node tree.Node) {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	curr, ok := dso.ships[name]
	if ok {
		delete(dso.ships, name)
		curr.Close()
	}
	dso.ships[name] = node
}

func (dso *shipsDso) Del(name string, node tree.Node) {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	curr, ok := dso.ships[name]
	if ok && curr.Name() == node.Name() {
		delete(dso.ships, name)
		curr.Close()
	}
}

func (dso *shipsDso) Count() int {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	return len(dso.ships)
}
