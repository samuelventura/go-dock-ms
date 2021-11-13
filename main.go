package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"

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

	rnode := tree.NewRoot("root", log.Println)
	defer rnode.WaitDisposed()
	//recover closes as well
	defer rnode.Recover()
	rnode.SetValue("source", getenv("DOCK_DB_SOURCE", withext("db3")))
	rnode.SetValue("driver", getenv("DOCK_DB_DRIVER", "sqlite"))
	dao := NewDao(rnode) //close on root
	rnode.AddCloser("dao", dao.Close)
	rnode.SetValue("dao", dao)
	for _, key := range dao.EnabledKeys() {
		log.Println("key", key.Name, strings.TrimSpace(key.Key))
	}
	dao.ClearShips()
	rnode.SetValue("ships", NewShips())

	spath := state.SingletonPath()
	snode := state.Serve(rnode, spath)
	defer snode.WaitDisposed()
	defer snode.Close()
	log.Println("socket", spath)

	enode := rnode.AddChild("ssh")
	defer enode.WaitDisposed()
	defer enode.Close()
	enode.SetValue("endpoint", getenv("DOCK_ENDPOINT_SSH", "0.0.0.0:31622"))
	enode.SetValue("hostkey", getenv("DOCK_HOSTKEY", withext("key")))
	enode.SetValue("maxships", getenvi("DOCK_MAXSHIPS", "1000"))
	enode.SetValue("export", getenv("DOCK_EXPORT_IP", "127.0.0.1"))
	sshd(enode)

	anode := rnode.AddChild("api")
	defer anode.WaitDisposed()
	defer anode.Close()
	anode.SetValue("endpoint", getenv("DOCK_ENDPOINT_API", "0.0.0.0:31623"))
	api(anode)

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
