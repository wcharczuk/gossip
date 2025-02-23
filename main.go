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
	gossipAddr = flag.String("gossip-addr", "gossip-srv:5050", "The gossip address")
	gossipName = flag.String("gossip-name", os.Getenv("HOSTNAME"), "The gossip name")
)

func main() {
	cfg := memberlist.DefaultLANConfig()
	cfg.Name = *gossipName
	list, err := memberlist.Create(cfg)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}
	_, err = list.Join([]string{*gossipAddr})
	if err != nil {
		panic("Failed to join cluster: " + err.Error())
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)
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
