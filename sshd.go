package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func sshd(args Args) *Result {
	dao := args.Get("dao").(Dao)
	host := args.Get("hostname").(string)
	endpoint := args.Get("endpoint").(string)
	hostkey := args.Get("hostkey").(string)
	maxships := args.Get("maxships").(int64)
	privateBytes, err := ioutil.ReadFile(hostkey)
	if err != nil {
		return &Result{err: err}
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return &Result{err: err}
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			inkey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
			dros, err := dao.GetKeys(host)
			if err != nil {
				return nil, err
			}
			for _, dro := range *dros {
				pubkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(dro.Key))
				if err != nil {
					log.Println("Ignoring invalid key", dro.Host, dro.Name)
					continue
				}
				pubtxt := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pubkey)))
				if pubtxt == inkey {
					return &ssh.Permissions{Extensions: map[string]string{"key-id": dro.Name}}, nil
				}
			}
			return nil, fmt.Errorf("key not found")
		},
	}
	config.AddHostKey(private)
	listen, err := net.Listen("tcp", endpoint)
	if err != nil {
		return &Result{err: err}
	}
	port := listen.Addr().(*net.TCPAddr).Port
	log.Println("port", port)
	args.Set("port", port)
	closer := NewCloser(func() {
		err := listen.Close()
		if err != nil {
			log.Println(err)
		}
	})
	args.Set("closer", closer)
	args.Set("config", config)
	go func() {
		for {
			tcpConn, err := listen.Accept()
			if err != nil {
				log.Println(err)
				closer.Close()
				return
			}
			count, err := dao.CountShips(host)
			if err != nil || count >= maxships {
				log.Println("max ships", maxships, count, err)
				tcpConn.Close()
				continue
			}
			go handleSshConnection(args.Clone(), tcpConn)
		}
	}()
	return closer.Result()
}

func handleSshConnection(args Args, tcpConn net.Conn) {
	defer tcpConn.Close()
	err := keepAlive(tcpConn)
	if err != nil {
		log.Println(err)
		return
	}
	dao := args.Get("dao").(Dao)
	host := args.Get("hostname").(string)
	closer := args.Get("closer").(*Closer)
	config := args.Get("config").(*ssh.ServerConfig)
	closed := make(chan interface{})
	defer close(closed)
	args.Set("closed", closed)
	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		log.Println(err)
		return
	}
	defer sshConn.Close()
	args.Set("ssh", sshConn)
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Println(err)
		return
	}
	defer listen.Close()
	ship := sshConn.User()
	port := listen.Addr().(*net.TCPAddr).Port
	log.Println(port, ship)
	args.Set("proxy", port)
	err = dao.AddEvent("open", host, ship, port)
	if err != nil {
		log.Println(err)
		closer.Close()
		return
	}
	err = dao.SetShip(host, ship, port)
	if err != nil {
		log.Println(err)
		closer.Close()
		return
	}
	defer func() {
		err := dao.ClearShip(host, port)
		if err != nil {
			log.Println(err)
			closer.Close()
		}
		err = dao.AddEvent("close", host, ship, port)
		if err != nil {
			log.Println(err)
			closer.Close()
		}
	}()
	go func() {
		defer listen.Close()
		select {
		case <-closed:
		case <-closer.Channel():
		}
	}()
	go func() {
		defer log.Println(port, "channel handler exited")
		defer listen.Close()
		for nch := range chans {
			nch.Reject(ssh.Prohibited, "unsupported")
		}
	}()
	go func() {
		defer log.Println(port, "request handler exited")
		defer listen.Close()
		for req := range reqs {
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}()
	go func() {
		defer log.Println(port, "ping handler exited")
		defer listen.Close()
		for {
			dl := time.Now().Add(10 * time.Second)
			resp, _, err := sshConn.SendRequest("ping", true, nil)
			if time.Now().After(dl) || err != nil || !resp {
				log.Println("ping timeout")
				return
			}
			timer := time.NewTimer(5 * time.Second)
			select {
			case <-timer.C:
				continue
			case <-closed:
				timer.Stop()
				return
			}
		}
	}()
	for {
		proxyConn, err := listen.Accept()
		if err != nil {
			log.Println(port, err)
			break
		}
		go handleProxyConnection(args.Clone(), proxyConn)
	}
}

func handleProxyConnection(args Args, proxyConn net.Conn) {
	defer proxyConn.Close()
	port := args.Get("proxy").(int)
	sshConn := args.Get("ssh").(*ssh.ServerConn)
	closed := args.Get("closed").(chan interface{})
	closer := args.Get("closer").(*Closer)
	err := keepAlive(proxyConn)
	if err != nil {
		log.Println(port, err)
		return
	}
	err = proxyConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		log.Println(port, err)
		return
	}
	var sb strings.Builder
	ba := make([]byte, 1)
	for {
		n, err := proxyConn.Read(ba)
		if err != nil {
			log.Println(port, err)
			return
		}
		if n != 1 {
			log.Println(port, fmt.Errorf("invalid read %d", n))
			return
		}
		err = sb.WriteByte(ba[0])
		if err != nil {
			log.Println(port, err)
			return
		}
		if ba[0] == 0x0A {
			break
		}
	}
	err = proxyConn.SetReadDeadline(time.Time{})
	if err != nil {
		log.Println(port, err)
		return
	}
	addr := strings.TrimSpace(sb.String())
	sshChan, reqChan, err := sshConn.OpenChannel("forward", []byte(addr))
	if err != nil {
		log.Println(port, err)
		return
	}
	defer sshChan.Close()
	go ssh.DiscardRequests(reqChan)
	done := make(chan interface{})
	go func() {
		_, err := io.Copy(sshChan, proxyConn)
		if err != nil {
			log.Println(port, err)
		}
		done <- true
	}()
	go func() {
		_, err := io.Copy(proxyConn, sshChan)
		if err != nil {
			log.Println(port, err)
		}
		done <- true
	}()
	select {
	case <-done:
	case <-closed:
	case <-closer.Channel():
	}
}
