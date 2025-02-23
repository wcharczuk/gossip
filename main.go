package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
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
	signal.Notify(shutdown, os.Interrupt)

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
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			fmt.Println("membership tick!")
			for _, member := range list.Members() {
				fmt.Printf("Member: %s %s\n", member.Name, member.Addr)
			}
		case <-shutdown:
			fmt.Println("shutting down!")
		}
	}
}
