package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/memberlist"
)

var (
	gossipAddr = flag.String("gossip-addr", "gossip-srv.gossip", "The gossip address")
)

func main() {
	flag.Parse()
	cfg := memberlist.DefaultLANConfig()
	cfg.Logger = log.New(io.Discard, "", 0)
	list, err := memberlist.Create(cfg)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

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
			var ips []net.IP
			ips, err = net.LookupIP(*gossipAddr)
			if err != nil {
				continue
			}
			var joinList []string
			for _, ip := range ips {
				joinList = append(joinList, ip.String())
			}
			fmt.Printf("Attempting to join %s based on DNS lookup.\n", strings.Join(joinList, ","))
			_, err = list.Join(joinList)
			if err != nil {
				continue
			}
			return
		}
	}
}

func runLoop(list *memberlist.Memberlist, shutdown chan os.Signal) {
	hostname, _ := os.Hostname()
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			fmt.Println(hostname, "membership tick!")
			if err := list.UpdateNode(10 * time.Second); err != nil {
				fmt.Fprintf(os.Stderr, "failed to update node: %v\n", err)
			}
			for _, member := range list.Members() {
				fmt.Printf("%s has Member: %s %s\n", hostname, member.Name, member.Addr)
			}
		case <-shutdown:
			fmt.Println(hostname, "shutting down!")
			if err := list.Leave(10 * time.Second); err != nil {
				fmt.Fprintf(os.Stderr, "failed to leave cluster: %v\n", err)
			}
			if err := list.Shutdown(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to shutdown: %v\n", err)
			}
			fmt.Println(hostname, "shutting complete")
			return
		}
	}
}
