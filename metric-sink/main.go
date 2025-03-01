package main

import (
	"context"
	"encoding/json"
	"gossip/pkg/types"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	dataMu sync.Mutex
	data   map[string]*metric
)

func bindAddr() string {
	if value := os.Getenv("BIND_ADDR"); value != "" {
		return value
	}
	return ":3000"
}

func main() {
	data = make(map[string]*metric)
	http.Handle("/", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		dataMu.Lock()
		defer dataMu.Unlock()
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(data)
	}))
	http.Handle("/submit", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var submissionData types.MetricSinkSubmission
		if err := json.NewDecoder(req.Body).Decode(&submissionData); err != nil {
			http.Error(rw, "invalid post body: "+err.Error(), http.StatusBadRequest)
			return
		}
		dataMu.Lock()
		defer dataMu.Unlock()
		for _, e := range submissionData.Values {
			if _, ok := data[e.Entity]; !ok {
				data[e.Entity] = new(metric)
			}
			data[e.Entity].Push(e)
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	server := &http.Server{
		Addr:    bindAddr(),
		Handler: http.DefaultServeMux,
	}
	shutdown := make(chan os.Signal, 3)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		timeoutContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(timeoutContext)
	}()
	server.ListenAndServe()
}

type metric struct {
	Last    time.Duration
	Count   uint64
	Values  []time.Duration
	Writers []string
}

func (m *metric) Push(s types.MetricSinkSubmissionValue) {
	m.Last = time.Duration(s.Value)
	m.Count++
	m.Values = append(m.Values, time.Duration(s.Value))
	m.Writers = append(m.Writers, s.Hostname)
}
