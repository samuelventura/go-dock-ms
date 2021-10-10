package main

import (
	"fmt"
	"time"

	"github.com/samuelventura/go-tree"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type daoDso struct {
	db *gorm.DB
}

type Dao interface {
	Close() error
	GetKeys(host string) (*[]KeyDro, error)
	AddKey(host, name, key string) error
	GetKey(host, name string) (*KeyDro, error)
	DelKey(host, name string) error
	CountShips(host string) (int64, error)
	ClearShips(host string) error
	ClearShip(host string, port int) error
	SetShip(host, name string, port int) error
	AddEvent(event, host, name string, port int) error
}

func dialector(node tree.Node) (gorm.Dialector, error) {
	driver := node.GetValue("driver").(string)
	source := node.GetValue("source").(string)
	switch driver {
	case "sqlite":
		return sqlite.Open(source), nil
	case "postgres":
		return postgres.Open(source), nil
	}
	return nil, fmt.Errorf("unknown driver %s", driver)
}

func NewDao(node tree.Node) (Dao, error) {
	mode := logger.Default.LogMode(logger.Silent)
	config := &gorm.Config{Logger: mode}
	dialector, err := dialector(node)
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(dialector, config)
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&KeyDro{}, &ShipDro{}, &LogDro{})
	if err != nil {
		return nil, err
	}
	return &daoDso{db}, nil
}

func (dso *daoDso) Close() error {
	sqlDB, err := dso.db.DB()
	if err != nil {
		return err
	}
	err = sqlDB.Close()
	if err != nil {
		return err
	}
	return nil
}

func (dso *daoDso) GetKeys(host string) (*[]KeyDro, error) {
	var dros []KeyDro
	result := dso.db.Where("host = ?", host).Find(&dros)
	return &dros, result.Error
}

func (dso *daoDso) AddKey(host, name, key string) error {
	dro := &KeyDro{Host: host, Name: name, Key: key}
	result := dso.db.Create(dro)
	return result.Error
}

func (dso *daoDso) GetKey(host, name string) (*KeyDro, error) {
	dro := &KeyDro{}
	result := dso.db.
		Where("host = ?", host).
		Where("name = ?", name).
		First(dro)
	return dro, result.Error
}

func (dso *daoDso) DelKey(host, name string) error {
	dro := &KeyDro{}
	result := dso.db.
		Where("host = ?", host).
		Where("name = ?", name).
		Delete(dro)
	if result.Error == nil && result.RowsAffected != 1 {
		return fmt.Errorf("row not found")
	}
	return result.Error
}

func (dso *daoDso) CountShips(host string) (int64, error) {
	var count int64
	dro := &ShipDro{}
	result := dso.db.Model(dro).
		Where("host = ?", host).
		Count(&count)
	return count, result.Error
}

func (dso *daoDso) ClearShips(host string) error {
	dro := &ShipDro{}
	result := dso.db.
		Where("host = ?", host).
		Delete(dro)
	return result.Error
}

func (dso *daoDso) SetShip(host, name string, port int) error {
	dro := &ShipDro{}
	dro.When = time.Now()
	dro.Host = host
	dro.Name = name
	dro.Port = port
	result := dso.db.Create(dro)
	return result.Error
}

func (dso *daoDso) ClearShip(host string, port int) error {
	dro := &ShipDro{}
	result := dso.db.
		Where("host = ?", host).
		Where("port = ?", port).
		Delete(dro)
	return result.Error
}

func (dso *daoDso) AddEvent(event, host, name string, port int) error {
	dro := &LogDro{}
	dro.Event = event
	dro.When = time.Now()
	dro.Host = host
	dro.Name = name
	dro.Port = port
	result := dso.db.Create(dro)
	return result.Error
}
