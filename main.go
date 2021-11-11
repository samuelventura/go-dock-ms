package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	"github.com/samuelventura/go-state"
	"github.com/samuelventura/go-tree"
)

func main() {
	os.Setenv("GOTRACEBACK", "all")
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	log.Println("start", os.Getpid())
	defer log.Println("exit")

	rlog := tree.NewLog()
	rnode := tree.NewRoot("root", rlog)
	defer rnode.WaitDisposed()
	//recover closes as well
	defer rnode.Recover()

	spath := state.SingletonPath()
	snode := state.Serve(rnode, spath)
	defer snode.WaitDisposed()
	defer snode.Close()
	log.Println("socket", spath)

	anode := rnode.AddChild("api")
	defer anode.WaitDisposed()
	defer anode.Close()
	host := hostname()
	anode.SetValue("hostname", getenv("DOCK_HOSTNAME", host))
	anode.SetValue("source", getenv("DOCK_DB_SOURCE", withext("db3")))
	anode.SetValue("driver", getenv("DOCK_DB_DRIVER", "sqlite"))
	anode.SetValue("endpoint", getenv("DOCK_ENDPOINT", "0.0.0.0:31622"))
	anode.SetValue("maxships", getenvi("DOCK_MAXSHIPS", "1000"))
	anode.SetValue("hostkey", getenv("DOCK_HOSTKEY", withext("key")))
	anode.SetValue("export", getenv("DOCK_EXPORT_IP", "127.0.0.1"))

	dao := NewDao(anode)
	anode.SetValue("dao", dao)
	defer dao.Close()
	dao.ClearShips(host)
	sshd(anode)

	stdin := make(chan interface{})
	go func() {
		defer close(stdin)
		ioutil.ReadAll(os.Stdin)
	}()
	select {
	case <-rnode.Closed():
	case <-snode.Closed():
	case <-anode.Closed():
	case <-ctrlc:
	case <-stdin:
	}
}
