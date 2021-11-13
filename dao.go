package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/samuelventura/go-tree"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type daoDso struct {
	mutex *sync.Mutex
	db    *gorm.DB
}

type Dao interface {
	Close() error
	ListKeys() []*KeyDro
	EnabledKeys() []*KeyDro
	GetKey(name string) (*KeyDro, error)
	AddKey(name, key string) error
	DelKey(name string) error
	EnableKey(name string, enabled bool) error
	ShipStart(sid, ship, key, host, ip string, port int)
	ShipStop(sid, ship, key, host, ip string, port int)
	ClearShips()
	CountShips() int64
	CountEnabledShips() int64
	CountDisabledShips() int64
	AddShip(name string) error
	GetShip(name string) (*ShipDro, error)
	EnableShip(name string, enabled bool) error
	PortShip(name string, port int) error
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
	err = db.AutoMigrate(&KeyDro{}, &ShipDro{}, &StateDro{}, &LogDro{})
	if err != nil {
		log.Fatal(err)
	}
	return &daoDso{&sync.Mutex{}, db}
}

func (dso *daoDso) Close() error {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
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
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	dro := &StateDro{}
	result := dso.db.Delete(dro, "true")
	if result.Error != nil {
		log.Fatal(result.Error)
	}
}

func (dso *daoDso) CountShips() int64 {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	count := int64(0)
	result := dso.db.Model(&ShipDro{}).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count
}

func (dso *daoDso) CountEnabledShips() int64 {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	count := int64(0)
	result := dso.db.Model(&ShipDro{}).Where("enabled = ?", true).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count
}

func (dso *daoDso) CountDisabledShips() int64 {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	count := int64(0)
	result := dso.db.Model(&ShipDro{}).Where("enabled != ?", true).Count(&count)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return count
}

func (dso *daoDso) AddShip(name string) error {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	dro := &ShipDro{Name: name}
	result := dso.db.Create(dro)
	return result.Error
}

func (dso *daoDso) GetShip(name string) (*ShipDro, error) {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	dro := &ShipDro{}
	result := dso.db.
		Where("name = ?", name).
		First(dro)
	return dro, result.Error
}

func (dso *daoDso) EnableShip(name string, enabled bool) error {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	result := dso.db.Model(&ShipDro{}).
		Where("name = ?", name).Update("enabled", enabled)
	if result.Error == nil && result.RowsAffected != 1 {
		return fmt.Errorf("ship not found")
	}
	return result.Error
}

func (dso *daoDso) PortShip(name string, port int) error {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	result := dso.db.Model(&ShipDro{}).
		Where("name = ?", name).Update("port", port)
	if result.Error == nil && result.RowsAffected != 1 {
		return fmt.Errorf("ship not found")
	}
	return result.Error
}

func (dso *daoDso) EnabledKeys() []*KeyDro {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	dros := []*KeyDro{}
	result := dso.db.Where("enabled", true).Find(&dros)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return dros
}

func (dso *daoDso) ListKeys() []*KeyDro {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	dros := []*KeyDro{}
	result := dso.db.Where("true").Find(&dros)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return dros
}

func (dso *daoDso) GetKey(name string) (*KeyDro, error) {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	dro := &KeyDro{}
	result := dso.db.
		Where("name = ?", name).
		First(dro)
	return dro, result.Error
}

func (dso *daoDso) AddKey(name, key string) error {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	dro := &KeyDro{Name: name, Key: key}
	result := dso.db.Create(dro)
	return result.Error
}

func (dso *daoDso) DelKey(name string) error {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	dro := &KeyDro{}
	result := dso.db.
		Where("name = ?", name).
		Delete(dro)
	if result.Error == nil && result.RowsAffected != 1 {
		return fmt.Errorf("key not found")
	}
	return result.Error
}

func (dso *daoDso) EnableKey(name string, enabled bool) error {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	result := dso.db.Model(&KeyDro{}).
		Where("name = ?", name).Update("enabled", enabled)
	if result.Error == nil && result.RowsAffected != 1 {
		return fmt.Errorf("key not found")
	}
	return result.Error
}

func (dso *daoDso) ShipStart(sid, ship, key, host, ip string, port int) {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	err := dso.addEvent(sid, "add", ship, key, host, ip, port)
	if err != nil {
		log.Fatal(err)
	}
	dro := &StateDro{}
	dro.Sid = sid
	dro.When = time.Now()
	dro.Ship = ship
	dro.Port = port
	dro.Host = host
	dro.IP = ip
	result := dso.db.Create(dro)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
}

func (dso *daoDso) ShipStop(sid, ship, key, host, ip string, port int) {
	dso.mutex.Lock()
	defer dso.mutex.Unlock()
	err := dso.addEvent(sid, "del", ship, key, host, ip, port)
	if err != nil {
		log.Fatal(err)
	}
	dro := &StateDro{}
	result := dso.db.Where("sid", sid).Delete(dro)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	if result.RowsAffected != 1 {
		log.Fatal("row not found")
	}
}

func (dso *daoDso) addEvent(sid, event, ship, key, host, ip string, port int) error {
	dro := &LogDro{}
	dro.Sid = sid
	dro.Event = event
	dro.When = time.Now()
	dro.Ship = ship
	dro.Key = key
	dro.Port = port
	dro.Host = host
	dro.IP = ip
	result := dso.db.Create(dro)
	return result.Error
}
