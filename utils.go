package main

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/felixge/tcpkeepalive"
)

func keepAlive(conn net.Conn) error {
	return tcpkeepalive.SetKeepAlive(
		conn, 5*time.Second, 3, 1*time.Second)
}

func getenv(name string, defval string) string {
	value := os.Getenv(name)
	if len(strings.TrimSpace(value)) > 0 {
		log.Println(name, value)
		return value
	}
	log.Println(name, defval)
	return defval
}

func getenvi(name string, defval int64) int64 {
	value := os.Getenv(name)
	if len(strings.TrimSpace(value)) > 0 {
		log.Println(name, value)
		val, err := strconv.Atoi(value)
		if err != nil {
			log.Fatal(err)
		}
		return int64(val)
	}
	log.Println(name, defval)
	return defval
}

func withext(ext string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	base := filepath.Base(exe)
	file := base + "." + ext
	return filepath.Join(dir, file), nil
}
