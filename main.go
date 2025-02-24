package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/memberlist"
)

var (
	gossipAddr = flag.String("gossip-addr", "gossip-srv", "The gossip address")
	gossipName = flag.String("gossip-name", os.Getenv("HOSTNAME"), "The gossip name")
)

func main() {
	cfg := memberlist.DefaultLANConfig()
	cfg.Name = *gossipName
	list, err := memberlist.Create(cfg)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	if err := tryJoin(list, shutdown); err != nil {
		panic("Failed to join memberlist: " + err.Error())
	}
	runLoop(list, shutdown)
}

func tryJoin(list *memberlist.Memberlist, shutdown chan os.Signal) (err error) {
	deadline := time.NewTimer(60 * time.Second)
	defer deadline.Stop()

	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for {
		select {
		case <-deadline.C:
			return fmt.Errorf("join deadline expired after 60s")
		case <-shutdown:
			return nil
		case <-tick.C:
			_, err = list.Join([]string{*gossipAddr})
			if err != nil {
				continue
			}
			return
		}
	}
}

func runLoop(list *memberlist.Memberlist, shutdown chan os.Signal) {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			fmt.Println(*gossipName, "membership tick!")
			if err := list.UpdateNode(10 * time.Second); err != nil {
				fmt.Fprintf(os.Stderr, "failed to update node: %v\n", err)
			}
			for _, member := range list.Members() {
				fmt.Printf(*gossipName, "Member: %s %s\n", member.Name, member.Addr)
			}
		case <-shutdown:
			fmt.Println(*gossipName, "shutting down!")
			if err := list.Leave(10 * time.Second); err != nil {
				fmt.Fprintf(os.Stderr, "failed to leave cluster: %v\n", err)
			}
			if err := list.Shutdown(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to shutdown: %v\n", err)
			}
			fmt.Println(*gossipName, "shutting complete")
		}
	}
}
