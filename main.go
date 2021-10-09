package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)

	log.Println("starting...")
	defer log.Println("exit")
	r := run(args())
	if r.err != nil {
		log.Fatal(r.err)
	}
	defer r.close()

	stdin := make(chan interface{})
	go func() {
		defer close(stdin)
		ioutil.ReadAll(os.Stdin)
	}()
	select {
	case <-r.closed:
	case <-ctrlc:
	case <-stdin:
	}
}

func args() Args {
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
	args := NewArgs()
	args.Set("hostname", getenv("DOCK_HOSTNAME", hostname))
	args.Set("source", getenv("DOCK_DB_SOURCE", source))
	args.Set("driver", getenv("DOCK_DB_DRIVER", "sqlite"))
	args.Set("endpoint", getenv("DOCK_ENDPOINT", "0.0.0.0:31652"))
	args.Set("maxships", getenvi("DOCK_MAXSHIPS", 1000))
	args.Set("hostkey", getenv("DOCK_HOSTKEY", hostkey))
	return args
}

func run(args Args) *Result {
	dao, err := NewDao(args)
	if err != nil {
		return &Result{err: err}
	}
	err = dao.ClearShips(args.Get("hostname").(string))
	if err != nil {
		return &Result{err: err}
	}
	args.Set("dao", dao)
	return sshd(args)
}
