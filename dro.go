package main

import "time"

type KeyDro struct {
	Name    string `gorm:"primaryKey"`
	Key     string
	Enabled bool
}

type ShipDro struct {
	Sid  string `gorm:"primaryKey"`
	Port int
	Ship string `gorm:"index"`
	When time.Time
}

type LogDro struct {
	Sid   string
	Event string
	Port  int
	Ship  string
	Key   string
	When  time.Time
}
