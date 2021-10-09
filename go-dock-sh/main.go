package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	log.Println("starting...")
	r := run(args())
	if r.err != nil {
		log.Fatal(r.err)
	}
	defer func() {
		log.Println("closing...")
		r.close()
		<-r.closed
		log.Println("closed")
	}()

	stdin := make(chan interface{})
	go func() {
		defer close(stdin)
		ioutil.ReadAll(os.Stdin)
	}()
	select {
	case <-r.closed:
	case <-ctrlc:
	case <-stdin:
	}
}

func args() Args {
	keypath, err := withext("key")
	if err != nil {
		log.Fatal(err)
	}
	iname, imac, err := nic()
	if err != nil {
		log.Fatal(err)
	}
	args := NewArgs()
	args.Set("keypath", getenv("DOCK_KEYPATH", keypath))
	args.Set("record", os.Getenv("DOCK_RECORD"))
	args.Set("nicname", iname)
	args.Set("nicmac", imac)
	return args
}

func run(args Args) *Result {
	record := args.Get("record").(string)
	keypath := args.Get("keypath").(string)
	key, err := ioutil.ReadFile(keypath)
	if err != nil {
		return &Result{err: err}
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return &Result{err: err}
	}
	user := "mac" + strings.ReplaceAll(args.Get("nicmac").(string), ":", "")
	hkcb := func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.HostKeyCallback(hkcb),
	}
	txts, err := net.LookupTXT(record)
	if err != nil {
		log.Println("record", record)
		return &Result{err: err}
	}
	for _, txt := range txts {
		addrs := strings.Split(txt, ",")
		l := len(addrs)
		n := rand.Intn(l)
		for i := 0; i < l; i++ {
			addr := addrs[(n+i)%l]
			addr = "localhost:31652"
			log.Println(addr, user)
			conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
			if err != nil {
				log.Println(err)
				continue
			}
			sshCon, sshch, reqch, err := ssh.NewClientConn(conn, addr, config)
			if err != nil {
				log.Println(err)
				conn.Close()
				continue
			}
			var closer = NewCloser(func() {
				err := sshCon.Close()
				if err != nil {
					log.Println(err)
				}
				err = conn.Close()
				if err != nil {
					log.Println(err)
				}
			})
			go func() {
				defer log.Println("request handler exited")
				for {
					timer := time.NewTimer(10 * time.Second)
					select {
					case req := <-reqch:
						if req != nil && req.Type == "ping" {
							err := req.Reply(true, nil)
							switch err {
							case nil:
								timer.Stop()
							default:
								closer.Close()
								return
							}
						}
					case <-timer.C:
						log.Println("idle timeout")
						closer.Close()
						return
					case <-closer.Channel():
						return
					}
				}
			}()
			handleForward := func(ch ssh.NewChannel) {
				addr := string(ch.ExtraData())
				log.Println("open", addr)
				defer log.Println("close", addr)
				sshch, _, err := ch.Accept()
				if err != nil {
					log.Println(err)
					return
				}
				defer sshch.Close()
				conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
				if err != nil {
					log.Println(err)
					return
				}
				defer conn.Close()
				done := make(chan interface{}, 2)
				go func() {
					io.Copy(sshch, conn)
					done <- true
				}()
				go func() {
					io.Copy(conn, sshch)
					done <- true
				}()
				select {
				case <-done: //close on first error
				case <-closer.Channel():
				}
			}
			go func() {
				defer log.Println("channel handler exited")
				defer closer.Close()
				for ch := range sshch {
					if ch.ChannelType() != "forward" {
						ch.Reject(ssh.Prohibited, "unsupported")
						return
					}
					go handleForward(ch)
				}
			}()
			return closer.Result()
		}
	}
	return &Result{err: fmt.Errorf("connection failed")}
}
