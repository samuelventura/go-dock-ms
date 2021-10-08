package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"

	"github.com/armon/go-socks5"
	"golang.org/x/crypto/ssh"
)

func sshd(args Args) (func(), error) {
	dao := args.Get("dao").(Dao)
	host := args.Get("hostname").(string)
	endpoint := args.Get("endpoint").(string)
	hostkey := args.Get("hostkey").(string)
	maxships := args.Get("maxships").(int64)
	privateBytes, err := ioutil.ReadFile(hostkey)
	if err != nil {
		return nil, err
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	port := listen.Addr().(*net.TCPAddr).Port
	log.Println("port", port)
	exit := make(chan interface{})
	go func() {
		for {
			tcpConn, err := listen.Accept()
			if err != nil {
				log.Println(err)
				close(exit)
				return
			}
			count, err := dao.CountShips(host)
			if err != nil || count >= maxships {
				log.Println("maxships", maxships, count, err)
				tcpConn.Close()
				continue
			}
			sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
			if err != nil {
				log.Println(err)
				continue
			}
			go func() {
				for newChannel := range chans {
					newChannel.Reject(ssh.Prohibited, "unsupported")
				}
			}()
			go ssh.DiscardRequests(reqs)
			go handleConnection(dao, host, sshConn)
		}
	}()
	closer := func() {
		err := listen.Close()
		if err != nil {
			log.Println(err)
		}
		<-exit
	}
	return closer, nil
}

func handleConnection(dao Dao, host string, sshConn *ssh.ServerConn) {
	defer sshConn.Close()
	ship := sshConn.User()
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Println(err)
		return
	}
	defer listen.Close()
	port := listen.Addr().(*net.TCPAddr).Port
	err = dao.AddEvent("open", host, ship, port)
	defer func() {
		err = dao.AddEvent("close", host, ship, port)
		if err != nil {
			log.Println(err)
		}
	}()
	if err != nil {
		log.Println(err)
		return
	}
	err = dao.SetShip(host, ship, port)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		err := dao.ClearShip(host, port)
		if err != nil {
			log.Println(err)
		}
	}()
	conf := &socks5.Config{
		Logger: log.Default(),
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			sshChan, reqChan, err := sshConn.OpenChannel("forward", []byte(addr))
			if err != nil {
				return nil, err
			}
			return &channelConn{sshChan, reqChan}, nil
		},
	}
	serverSocks, err := socks5.New(conf)
	if err != nil {
		log.Println(err)
		return
	}
	ping := make(chan interface{})
	go func() {
		err = serverSocks.Serve(listen)
		if err != nil {
			ping <- false
			return
		}
	}()
	go func() {
		for {
			resp, _, err := sshConn.SendRequest("ping", true, nil)
			if err != nil {
				ping <- false
				return
			}
			if resp {
				ping <- true
				log.Println("pong")
				time.Sleep(5 * time.Second)
			}
		}
	}()
	for {
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-timer.C:
			return
		case val := <-ping:
			timer.Stop()
			if !val.(bool) {
				return
			}
		}
	}
}

type channelConn struct {
	sshch ssh.Channel
	reqch <-chan *ssh.Request
}

func (cc *channelConn) Read(b []byte) (n int, err error) {
	return cc.sshch.Read(b)
}

func (cc *channelConn) Write(b []byte) (n int, err error) {
	return cc.sshch.Write(b)
}

func (cc *channelConn) Close() error {
	return cc.sshch.Close()
}

func (cc *channelConn) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (cc *channelConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

func (cc *channelConn) SetDeadline(t time.Time) error {
	return nil
}

func (cc *channelConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (cc *channelConn) SetWriteDeadline(t time.Time) error {
	return nil
}
