package main

import (
	"fmt"
	"log"
)

type argsDso struct {
	values map[string]interface{}
}

type Args interface {
	Clone() Args
	Set(name string, value interface{})
	Get(name string) interface{}
}

func NewArgs() Args {
	dso := &argsDso{}
	dso.values = make(map[string]interface{})
	return dso
}

func (dso *argsDso) Clone() Args {
	ndso := &argsDso{}
	ndso.values = make(map[string]interface{})
	for k, v := range dso.values {
		ndso.values[k] = v
	}
	return ndso
}

func (dso *argsDso) Get(name string) interface{} {
	value, ok := dso.values[name]
	if !ok {
		err := fmt.Errorf("args value not found %s", name)
		log.Fatal(err)
	}
	return value
}

func (dso *argsDso) Set(name string, value interface{}) {
	dso.values[name] = value
}
