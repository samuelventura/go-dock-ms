package main

import (
	"fmt"
	"log"
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
	ClearShips()
	GetKeys() *[]KeyDro
	AddKey(name, key string) error
	GetKey(name string) (*KeyDro, error)
	DelKey(name string) error
	AddShip(sid, ship, key string, port int)
	DelShip(sid, ship, key string, port int)
}

func dialector(node tree.Node) gorm.Dialector {
	driver := node.GetValue("driver").(string)
	source := node.GetValue("source").(string)
	switch driver {
	case "sqlite":
		return sqlite.Open(source)
	case "postgres":
		return postgres.Open(source)
	}
	log.Fatalf("unknown driver %s", driver)
	return nil
}

func NewDao(node tree.Node) Dao {
	mode := logger.Default.LogMode(logger.Silent)
	config := &gorm.Config{Logger: mode}
	dialector := dialector(node)
	db, err := gorm.Open(dialector, config)
	if err != nil {
		log.Fatal(err)
	}
	err = db.AutoMigrate(&KeyDro{}, &ShipDro{}, &LogDro{})
	if err != nil {
		log.Fatal(err)
	}
	return &daoDso{db}
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

func (dso *daoDso) ClearShips() {
	dro := &ShipDro{}
	result := dso.db.Delete(dro, "true")
	if result.Error != nil {
		log.Fatal(result.Error)
	}
}

func (dso *daoDso) GetKeys() *[]KeyDro {
	dros := []KeyDro{}
	result := dso.db.Where("enabled", true).Find(&dros)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return &dros
}

func (dso *daoDso) AddKey(name, key string) error {
	dro := &KeyDro{Name: name, Key: key}
	result := dso.db.Create(dro)
	return result.Error
}

func (dso *daoDso) GetKey(name string) (*KeyDro, error) {
	dro := &KeyDro{}
	result := dso.db.
		Where("name = ?", name).
		First(dro)
	return dro, result.Error
}

func (dso *daoDso) DelKey(name string) error {
	dro := &KeyDro{}
	result := dso.db.
		Where("name = ?", name).
		Delete(dro)
	if result.Error == nil && result.RowsAffected != 1 {
		return fmt.Errorf("row not found")
	}
	return result.Error
}

func (dso *daoDso) AddShip(sid, ship, key string, port int) {
	err := dso.addEvent(sid, "add", ship, key, port)
	if err != nil {
		log.Fatal(err)
	}
	dro := &ShipDro{}
	dro.Sid = sid
	dro.When = time.Now()
	dro.Ship = ship
	dro.Port = port
	result := dso.db.Create(dro)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
}

func (dso *daoDso) DelShip(sid, ship, key string, port int) {
	err := dso.addEvent(sid, "del", ship, key, port)
	if err != nil {
		log.Fatal(err)
	}
	dro := &ShipDro{}
	result := dso.db.Where("sid", sid).Delete(dro)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	if result.RowsAffected != 1 {
		log.Fatal("row not found")
	}
}

func (dso *daoDso) addEvent(sid, event, ship, key string, port int) error {
	dro := &LogDro{}
	dro.Sid = sid
	dro.Event = event
	dro.When = time.Now()
	dro.Ship = ship
	dro.Key = key
	dro.Port = port
	result := dso.db.Create(dro)
	return result.Error
}
