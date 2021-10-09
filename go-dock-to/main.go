package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"golang.org/x/net/proxy"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	proxyurl := os.Args[1]
	address := os.Args[2]

	dialer, err := proxy.SOCKS5("tcp", proxyurl, nil,
		&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)
	if err != nil {
		log.Println(err)
		return
	}
	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		log.Println(err)
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
