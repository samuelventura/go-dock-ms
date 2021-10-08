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
	defer log.Println("exit")
	closer, err := run(args())
	if err != nil {
		log.Fatal(err)
	}
	defer closer()

	exit := make(chan interface{})
	go func() {
		defer close(exit)
		ioutil.ReadAll(os.Stdin)
	}()
	select {
	case <-ctrlc:
	case <-exit:
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

func run(args Args) (func(), error) {
	record := args.Get("record").(string)
	keypath := args.Get("keypath").(string)
	key, err := ioutil.ReadFile(keypath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
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
		return nil, err
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
			go func() {
				for req := range reqch {
					if req.Type == "ping" {
						req.Reply(true, nil)
					}
				}
			}()
			handleForward := func(ch ssh.NewChannel) {
				addr := string(ch.ExtraData())
				log.Println("forward", addr)
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
				go func() { io.Copy(sshch, conn) }()
				io.Copy(conn, sshch)
			}
			go func() {
				for ch := range sshch {
					if ch.ChannelType() != "forward" {
						ch.Reject(ssh.Prohibited, "unsupported")
					}
					go handleForward(ch)
				}
			}()
			return func() {
				err := sshCon.Close()
				if err != nil {
					log.Println(err)
				}
				err = conn.Close()
				if err != nil {
					log.Println(err)
				}
			}, nil
		}
	}
	return nil, fmt.Errorf("connection failed")
}
