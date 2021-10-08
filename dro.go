package main

import "time"

type KeyDro struct {
	Name string `gorm:"primaryKey"`
	Host string `gorm:"primaryKey"`
	Key  string `gorm:"index"`
}

type ShipDro struct {
	Port int    `gorm:"primaryKey"`
	Host string `gorm:"primaryKey"`
	Name string `gorm:"index"`
	When time.Time
}

type LogDro struct {
	Event string
	Port  int
	Host  string
	Name  string
	When  time.Time
}
