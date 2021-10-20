package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"

	"github.com/samuelventura/go-tree"
	"golang.org/x/crypto/ssh"
)

func sshd(node tree.Node) error {
	dao := node.GetValue("dao").(Dao)
	host := node.GetValue("hostname").(string)
	endpoint := node.GetValue("endpoint").(string)
	hostkey := node.GetValue("hostkey").(string)
	maxships := node.GetValue("maxships").(int64)
	privateBytes, err := ioutil.ReadFile(hostkey)
	if err != nil {
		return err
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return err
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
	node.SetValue("config", config)
	listen, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	node.AddCloser("listen", listen.Close)
	port := listen.Addr().(*net.TCPAddr).Port
	log.Println("port", port)
	node.SetValue("port", port)
	node.AddProcess("listen", func() {
		id := NewId("ssh-" + listen.Addr().String())
		for {
			tcpConn, err := listen.Accept()
			if err != nil {
				log.Println(err)
				return
			}
			count, err := dao.CountShips(host)
			if err != nil {
				log.Fatalln(err)
				return
			}
			if count >= maxships {
				log.Println("max ships", maxships, count, err)
				tcpConn.Close()
				continue
			}
			addr := tcpConn.RemoteAddr().String()
			cid := id.Next(addr)
			child := node.AddChild(cid)
			if child == nil {
				tcpConn.Close()
				continue
			}
			child.AddCloser("tcpConn", tcpConn.Close)
			child.AddProcess("tcpConn", func() {
				handleSshConnection(child, tcpConn)
			})
		}
	})
	return nil
}

func handleSshConnection(node tree.Node, tcpConn net.Conn) {
	err := keepAlive(tcpConn)
	if err != nil {
		log.Println(err)
		return
	}
	dao := node.GetValue("dao").(Dao)
	host := node.GetValue("hostname").(string)
	config := node.GetValue("config").(*ssh.ServerConfig)
	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		log.Println(err)
		return
	}
	node.AddCloser("sshConn", sshConn.Close)
	node.SetValue("ssh", sshConn)
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Println(err)
		return
	}
	node.AddCloser("listen", listen.Close)
	ship := sshConn.User()
	port := listen.Addr().(*net.TCPAddr).Port
	log.Println(port, ship)
	node.SetValue("proxy", port)
	err = dao.AddEvent("open", host, ship, port)
	if err != nil {
		log.Fatalln(err)
		return
	}
	err = dao.SetShip(host, ship, port)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer func() {
		err := dao.ClearShip(host, port)
		if err != nil {
			log.Fatalln(err)
		}
		err = dao.AddEvent("close", host, ship, port)
		if err != nil {
			log.Fatalln(err)
		}
	}()
	node.AddProcess("ssh chans reject", func() {
		for nch := range chans {
			nch.Reject(ssh.Prohibited, "unsupported")
		}
	})
	node.AddProcess("ssh reqs reply", func() {
		for req := range reqs {
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	})
	node.AddProcess("ssh ping handler", func() {
		for {
			dl := time.Now().Add(10 * time.Second)
			resp, _, err := sshConn.SendRequest("ping", true, nil)
			if time.Now().After(dl) || err != nil || !resp {
				log.Println(port, "ping timeout")
				return
			}
			timer := time.NewTimer(5 * time.Second)
			select {
			case <-timer.C:
				continue
			case <-node.Closed():
				timer.Stop()
				return
			}
		}
	})
	id := NewId("proxy-" + listen.Addr().String())
	for {
		proxyConn, err := listen.Accept()
		if err != nil {
			log.Println(port, err)
			break
		}
		addr := proxyConn.RemoteAddr().String()
		cid := id.Next(addr)
		child := node.AddChild(cid)
		if child == nil {
			proxyConn.Close()
			continue
		}
		child.AddCloser("proxyConn", proxyConn.Close)
		child.AddProcess("proxyConn", func() {
			handleProxyConnection(child, proxyConn)
		})
	}
}

func handleProxyConnection(node tree.Node, proxyConn net.Conn) {
	defer node.Close()
	port := node.GetValue("proxy").(int)
	sshConn := node.GetValue("ssh").(*ssh.ServerConn)
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
	node.AddCloser("sshChan", sshChan.Close)
	node.AddProcess("DiscardRequests(reqChan)", func() { ssh.DiscardRequests(reqChan) })
	node.AddProcess("Copy(sshChan, proxyConn)", func() {
		_, err := io.Copy(sshChan, proxyConn)
		if err != nil {
			log.Println(port, err)
		}
	})
	node.AddProcess("Copy(proxyConn, sshChan)", func() {
		_, err := io.Copy(proxyConn, sshChan)
		if err != nil {
			log.Println(port, err)
		}
	})
	<-node.Closed()
}
