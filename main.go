package main

import (
	"log"
	"os"
	"strings"

	"github.com/samuelventura/go-state"
	"github.com/samuelventura/go-tools"
	"github.com/samuelventura/go-tree"
)

func main() {
	tools.SetupLog()

	ctrlc := tools.SetupCtrlc()
	stdin := tools.SetupStdinAll()

	log.Println("start", os.Getpid())
	defer log.Println("exit")

	rnode := tree.NewRoot("root", log.Println)
	defer rnode.WaitDisposed()
	//recover closes as well
	defer rnode.Recover()
	rnode.SetValue("hostname", tools.GetHostname())
	rnode.SetValue("source", tools.GetEnviron("DOCK_DB_SOURCE", tools.WithExtension("db3")))
	rnode.SetValue("driver", tools.GetEnviron("DOCK_DB_DRIVER", "sqlite"))
	rnode.SetValue("state", tools.GetEnviron("DOCK_STATE", tools.WithExtension("state")))
	dao := NewDao(rnode) //close on root
	rnode.AddCloser("dao", dao.Close)
	rnode.SetValue("dao", dao)
	for _, key := range dao.EnabledKeys() {
		log.Println("key", key.Name, strings.TrimSpace(key.Key))
	}
	dao.ClearShips()
	rnode.SetValue("ships", NewShips())

	snode := state.Serve(rnode, rnode.GetValue("state").(string))
	defer snode.WaitDisposed()
	defer snode.Close()

	enode := rnode.AddChild("ssh")
	defer enode.WaitDisposed()
	defer enode.Close()
	enode.SetValue("endpoint", tools.GetEnviron("DOCK_ENDPOINT_SSH", "0.0.0.0:31622"))
	enode.SetValue("hostkey", tools.GetEnviron("DOCK_HOSTKEY", tools.WithExtension("key")))
	enode.SetValue("maxships", tools.GetEnvironInt("DOCK_MAXSHIPS", 10, 32, 1000))
	enode.SetValue("export", tools.GetEnviron("DOCK_EXPORT_IP", "127.0.0.1"))
	sshd(enode)

	anode := rnode.AddChild("api")
	defer anode.WaitDisposed()
	defer anode.Close()
	anode.SetValue("endpoint", tools.GetEnviron("DOCK_ENDPOINT_API", "127.0.0.1:31623"))
	api(anode)

	select {
	case <-rnode.Closed():
	case <-snode.Closed():
	case <-enode.Closed():
	case <-anode.Closed():
	case <-ctrlc:
	case <-stdin:
	}
}
