package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	"github.com/samuelventura/go-tree"
)

func main() {
	os.Setenv("GOTRACEBACK", "all")
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	log.Println(os.Getpid(), "starting...")
	defer log.Println("exit")
	node := root()
	defer node.WaitDisposed()
	defer node.Close()
	err := run(node)
	if err != nil {
		log.Fatal(err)
	}

	stdin := make(chan interface{})
	go func() {
		defer close(stdin)
		ioutil.ReadAll(os.Stdin)
	}()
	select {
	case <-node.Closed():
		log.Println("root closed")
	case <-ctrlc:
		log.Println("ctrlc interrupt")
	case <-stdin:
		log.Println("stdin closed")
	}
}

func root() tree.Node {
	source, err := withext("db3")
	if err != nil {
		log.Fatal(err)
	}
	hostkey, err := withext("key")
	if err != nil {
		log.Fatal(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	node := tree.NewRoot(nil)
	node.SetValue("hostname", getenv("DOCK_HOSTNAME", hostname))
	node.SetValue("source", getenv("DOCK_DB_SOURCE", source))
	node.SetValue("driver", getenv("DOCK_DB_DRIVER", "sqlite"))
	node.SetValue("endpoint", getenv("DOCK_ENDPOINT", "0.0.0.0:31652"))
	node.SetValue("maxships", getenvi("DOCK_MAXSHIPS", 1000))
	node.SetValue("hostkey", getenv("DOCK_HOSTKEY", hostkey))
	return node
}

func run(node tree.Node) error {
	dao, err := NewDao(node)
	if err != nil {
		return err
	}
	node.SetValue("dao", dao)
	hostname := node.GetValue("hostname").(string)
	err = dao.ClearShips(hostname)
	if err != nil {
		return err
	}
	return sshd(node)
}
