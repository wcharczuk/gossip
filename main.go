package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"gossip/pkg/consistenthash"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/memberlist"
)

var (
	gossipAddr = flag.String("gossip-addr", "gossip-members.gossip", "The gossip address")
)

func main() {
	flag.Parse()
	cfg := memberlist.DefaultLANConfig()
	cfg.Logger = log.New(io.Discard, "", 0)

	shutdown := make(chan os.Signal, 3)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	w := &worker{
		shutdown: shutdown,
	}
	w.hostname, _ = os.Hostname()
	cfg.Events = w

	list, err := memberlist.Create(cfg)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}
	w.list = list
	if err := w.tryJoin(); err != nil {
		panic("Failed to join memberlist: " + err.Error())
	}
	if err := w.runLoop(); err != nil {
		panic("Worker failure: " + err.Error())
	}
}

type worker struct {
	memberlist.EventDelegate
	hostname string
	list     *memberlist.Memberlist
	shutdown <-chan os.Signal
}

func (w worker) NotifyJoin(n *memberlist.Node) {
	slog.Info("node joined", slog.String("name", n.Name))
}

func (w worker) NotifyLeave(n *memberlist.Node) {
	slog.Info("node left", slog.String("name", n.Name))
}

func (w worker) NotifyUpdate(n *memberlist.Node) {
	slog.Info("node update", slog.String("name", n.Name))
}

func (w worker) tryJoin() (err error) {
	deadline := time.NewTimer(60 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-deadline.C:
			return fmt.Errorf("join deadline expired after 60s")
		case <-w.shutdown:
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
			_, err = w.list.Join(joinList)
			if err != nil {
				continue
			}
			return
		}
	}
}

func (w worker) runLoop() error {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			entities, err := w.getEntityList()
			if err != nil {
				slog.Error("failed to get entities", slog.Any("err", err))
				continue
			}
			members := w.getMembers()
			ch := consistenthash.New()
			ch.AddBuckets(members...)
			var matchedEntities []string
			for _, e := range entities {
				if ch.Assignment(e) == w.hostname {
					matchedEntities = append(matchedEntities, e)
				}
			}

		case <-w.shutdown:
			w.doShutdown()
			return nil
		}
	}
}

func (w worker) getEntityList() (entities []string, err error) {
	var res *http.Response
	res, err = http.Get("data-plane:3000/")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&entities)
	return
}

func (w worker) getAndPushEntities(entities ...string) error {
	data, err := w.getEntityData(entities...)
	if err != nil {
		return err
	}
	return w.pushEntities(data)
}

func (w worker) getEntityData(entities ...string) (data map[string]uint64, err error) {
	u, _ := url.Parse("http://data-plane:3000/data")
	u.RawQuery = fmt.Sprintf("s=%s", strings.Join(entities, ","))
	var res *http.Response
	res, err = http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&entities)
	return
}

type Submission struct {
	Values []EntityValue
}

type EntityValue struct {
	Entity   string
	Hostname string
	Value    uint64
}

func (w worker) pushEntities(values map[string]uint64) error {
	var submission Submission
	for key, value := range values {
		submission.Values = append(submission.Values, EntityValue{
			Entity:   key,
			Hostname: w.hostname,
			Value:    value,
		})
	}
	body, err := json.Marshal(submission)
	if err != nil {
		return err
	}
	_, err = http.Post("http://metric-sink:3000/submit", "application/json", bytes.NewReader(body))
	return err
}

func (w worker) getMembers() (memberNames []string) {
	members := w.list.Members()
	slices.SortFunc(members, func(i, j *memberlist.Node) int {
		if i.Name < j.Name {
			return -1
		}
		if i.Name == j.Name {
			return 0
		}
		return 1
	})
	for _, m := range members {
		memberNames = append(memberNames, m.Name)
	}
	return
}

func (w worker) doShutdown() {
	fmt.Println(w.hostname, "shutting down!")
	if err := w.list.Leave(10 * time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to leave cluster: %v\n", w.hostname, err)
	}
	if err := w.list.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "%s failed to shutdown: %v\n", w.hostname, err)
	}
	fmt.Println(w.hostname, "shutting complete")
}
