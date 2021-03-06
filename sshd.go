package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"

	"github.com/samuelventura/go-tools"
	"github.com/samuelventura/go-tree"
	"golang.org/x/crypto/ssh"
)

func sshd(node tree.Node) {
	dao := node.GetValue("dao").(Dao)
	ships := node.GetValue("ships").(Ships)
	hostname := node.GetValue("hostname").(string)
	endpoint := node.GetValue("endpoint").(string)
	hostkey := node.GetValue("hostkey").(string)
	maxships := node.GetValue("maxships").(int64)
	privateBytes, err := ioutil.ReadFile(hostkey)
	if err != nil {
		log.Panicln(err)
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Panicln(err)
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			inkey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
			for _, key := range dao.EnabledKeys() {
				pubkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.Key))
				if err != nil {
					log.Panicln("Ignoring invalid key", key.Name)
				}
				pubtxt := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pubkey)))
				if pubtxt == inkey {
					return &ssh.Permissions{Extensions: map[string]string{"key-id": key.Name}}, nil
				}
			}
			return nil, fmt.Errorf("key not found")
		},
	}
	config.AddHostKey(private)
	node.SetValue("config", config)
	listen, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Panicln(err)
	}
	node.AddCloser("listen", listen.Close)
	port := listen.Addr().(*net.TCPAddr).Port
	log.Println("port ssh", port)
	node.SetValue("port", port)
	node.AddProcess("listen", func() {
		id := NewId("ssh-" + hostname + "-" + listen.Addr().String())
		for {
			tcpConn, err := listen.Accept()
			if err != nil {
				log.Println(err)
				return
			}
			count := int64(ships.Count())
			if count >= maxships {
				log.Println("max ships", maxships, count)
				tcpConn.Close()
				continue
			}
			setupSshConnection(node, tcpConn, ships, id)
		}
	})
}

func setupSshConnection(node tree.Node, tcpConn net.Conn, ships Ships, id Id) {
	defer node.IfRecoverCloser(tcpConn.Close)
	addr := tcpConn.RemoteAddr().String()
	cid := id.Next(addr)
	child := node.AddChild(cid)
	child.AddCloser("tcpConn", tcpConn.Close)
	child.AddProcess("tcpConn", func() {
		handleSshConnection(child, tcpConn, ships)
	})
}

func handleSshConnection(node tree.Node, tcpConn net.Conn, ships Ships) {
	tools.KeepAlive(tcpConn, 5)
	dao := node.GetValue("dao").(Dao)
	export := node.GetValue("export").(string)
	hostname := node.GetValue("hostname").(string)
	config := node.GetValue("config").(*ssh.ServerConfig)
	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		log.Println(err)
		return
	}
	ship := sshConn.User()
	dro, err := dao.GetShip(ship)
	if err != nil || !dro.Enabled {
		log.Println(ship, dro.Enabled, err)
		return
	}
	node.AddCloser("sshConn", sshConn.Close)
	node.SetValue("ssh", sshConn)
	endpoint := fmt.Sprintf("%s:%d", export, dro.Port)
	listen, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Println(err)
		return
	}
	node.AddCloser("listen", listen.Close)
	port := listen.Addr().(*net.TCPAddr).Port
	key := sshConn.Permissions.Extensions["key-id"]
	node.SetValue("proxy", port)
	node.SetValue("key", key)
	//replace ship by name, ensure sport already defined
	ships.Add(ship, node)
	defer ships.Del(ship, node)
	log.Println(ship, port, tcpConn.RemoteAddr(), ships.Count())
	dao.ShipStart(node.Name(), ship, key, hostname, export, port)
	defer dao.ShipStop(node.Name(), ship, key, hostname, export, port)
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
		setupProxyConnection(node, proxyConn, id)
	}
}

func setupProxyConnection(node tree.Node, proxyConn net.Conn, id Id) {
	defer node.IfRecoverCloser(proxyConn.Close)
	addr := proxyConn.RemoteAddr().String()
	cid := id.Next(addr)
	child := node.AddChild(cid)
	child.AddCloser("proxyConn", proxyConn.Close)
	child.AddProcess("proxyConn", func() {
		handleProxyConnection(child, proxyConn)
	})
}

func handleProxyConnection(node tree.Node, proxyConn net.Conn) {
	tools.KeepAlive(proxyConn, 5)
	port := node.GetValue("proxy").(int)
	sshConn := node.GetValue("ssh").(*ssh.ServerConn)
	err := proxyConn.SetReadDeadline(time.Now().Add(5 * time.Second))
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
	node.AddProcess("DiscardRequests(reqChan)", func() {
		ssh.DiscardRequests(reqChan)
	})
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
	node.WaitClosed()
}
