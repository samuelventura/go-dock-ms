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
	"github.com/felixge/tcpkeepalive"
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
	closer := NewCloser(func() {
		err := listen.Close()
		if err != nil {
			log.Println(err)
		}
	})
	args.Set("parent", closer)
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
			go handleConnection(args, tcpConn)
		}
	}()
	return closer.Result()
}

func handleConnection(args Args, tcpConn net.Conn) {
	dao := args.Get("dao").(Dao)
	host := args.Get("hostname").(string)
	parent := args.Get("parent").(*Closer)
	config := args.Get("config").(*ssh.ServerConfig)
	closed := make(chan interface{})
	defer close(closed)
	defer tcpConn.Close()
	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		log.Println(err)
		return
	}
	defer sshConn.Close()
	ship := sshConn.User()
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Println(err)
		return
	}
	defer listen.Close()
	port := listen.Addr().(*net.TCPAddr).Port
	log.Println(port, ship)
	err = dao.AddEvent("open", host, ship, port)
	if err != nil {
		log.Println(err)
		parent.Close()
		return
	}
	err = dao.SetShip(host, ship, port)
	if err != nil {
		log.Println(err)
		parent.Close()
		return
	}
	defer func() {
		err := dao.ClearShip(host, port)
		if err != nil {
			log.Println(err)
			parent.Close()
		}
		err = dao.AddEvent("close", host, ship, port)
		if err != nil {
			log.Println(err)
			parent.Close()
		}
	}()
	go func() {
		defer listen.Close()
		select {
		case <-closed:
		case <-parent.Channel():
		}
	}()
	conf := &socks5.Config{
		//FIXME nop logger
		Logger: log.Default(),
		//FIXME close on context
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			log.Println(port, "dialing", network, addr)
			sshChan, reqChan, err := sshConn.OpenChannel("forward", []byte(addr))
			if err != nil {
				log.Println(port, "dial", err)
				return nil, err
			}
			return &channelConn{sshChan, reqChan}, nil
		},
	}
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
		for {
			dl := time.Now().Add(10 * time.Second)
			resp, _, err := sshConn.SendRequest("ping", true, nil)
			if time.Now().After(dl) || err != nil || !resp {
				log.Println("ping timeout")
				listen.Close()
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
	server, err := socks5.New(conf)
	if err != nil {
		log.Println(port, err)
		return
	}
	err = server.Serve(&Listener{listen})
	if err != nil {
		log.Println(port, err)
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
	//FIXME implement keepalive
	log.Println("closing connection...")
	return cc.sshch.Close()
}

func (cc *channelConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0}
}

func (cc *channelConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0}
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

type Listener struct {
	listener net.Listener
}

func (l *Listener) Accept() (net.Conn, error) {
	c, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}
	err = tcpkeepalive.SetKeepAlive(c, 5*time.Second, 3, 1*time.Second)
	if err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func (l *Listener) Close() error {
	return l.listener.Close()
}

func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}
