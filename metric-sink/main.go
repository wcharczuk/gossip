package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
)

var (
	dataMu sync.Mutex
	data   map[string]*Metric
)

func bindAddr() string {
	if value := os.Getenv("BIND_ADDR"); value != "" {
		return value
	}
	return ":3000"
}

func main() {
	http.Handle("/", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		dataMu.Lock()
		defer dataMu.Unlock()
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(data)
	}))
	http.Handle("/submit", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var submissionData Submission
		_ = json.NewDecoder(req.Body).Decode(&submissionData)
		for _, e := range submissionData.Values {
			if _, ok := data[e.Entity]; !ok {
				dataMu.Lock()
				data[e.Entity] = new(Metric)
				dataMu.Unlock()
			}
			data[e.Entity].Push(e)
		}
	}))
	http.ListenAndServe(bindAddr(), nil)
}

type Submission struct {
	Values []EntityValue
}

type EntityValue struct {
	Entity   string
	Hostname string
	Value    uint64
}

type Metric struct {
	sync.Mutex `json:"-"`
	Last       uint64
	Count      uint64
	Values     []uint64
	Writers    []string
}

func (m *Metric) Push(s EntityValue) {
	m.Lock()
	defer m.Unlock()
	m.Last = s.Value
	m.Count++
	m.Values = append(m.Values, s.Value)
	m.Writers = append(m.Writers, s.Hostname)
}
