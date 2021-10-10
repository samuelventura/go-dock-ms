package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	proxyurl := os.Args[1]
	address := os.Args[2]

	conn, err := net.DialTimeout("tcp", proxyurl, 5*time.Second)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	line := fmt.Sprintln(address)
	n, err := conn.Write([]byte(line))
	if err != nil {
		log.Println(err)
		return
	}
	if n != len(line) {
		log.Println(fmt.Errorf("write mismatch %d %d", len(line), n))
		return
	}
	done := make(chan interface{})
	go func() {
		io.Copy(os.Stdout, conn)
		done <- true
	}()
	go func() {
		io.Copy(conn, os.Stdin)
		done <- true
	}()
	select {
	case <-ctrlc:
	case <-done:
	}
}
