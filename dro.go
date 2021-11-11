package main

import "time"

type KeyDro struct {
	Host string `gorm:"primaryKey"`
	Name string `gorm:"primaryKey"`
	Key  string `gorm:"index"`
}

type ShipDro struct {
	Host string `gorm:"primaryKey"`
	Port int    `gorm:"primaryKey"`
	Ship string `gorm:"index"`
	When time.Time
}

type LogDro struct {
	Event string
	Port  int
	Host  string
	Ship  string
	When  time.Time
}
