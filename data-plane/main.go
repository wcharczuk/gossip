package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	_ "embed"
)

//go:embed symbols.txt
var symbolsData []byte

var (
	entities map[string]*Entity
)

func bindAddr() string {
	if value := os.Getenv("BIND_ADDR"); value != "" {
		return value
	}
	return ":3000"
}

func main() {
	symbols := getSymbols()
	entities = make(map[string]*Entity, len(symbols))
	for _, symbol := range symbols {
		entities[symbol] = &Entity{0}
	}
	http.Handle("/", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		response := make([]string, 0, len(entities))
		for key := range entities {
			response = append(response, key)
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(response)
	}))
	http.Handle("/data", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var requestSymbols []string
		if rawRequestSymbols := req.URL.Query().Get("s"); rawRequestSymbols != "" {
			requestSymbols = strings.Split(rawRequestSymbols, ",")
		}
		response := Response{
			Entities: make(map[string]uint64),
		}
		if len(requestSymbols) > 0 {
			for _, requestSymbol := range requestSymbols {
				response.Entities[requestSymbol] = entities[requestSymbol].Increment()
			}
		} else {
			for symbol, entity := range entities {
				response.Entities[symbol] = entity.Increment()
			}
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(response)
	}))
	_ = http.ListenAndServe(bindAddr(), nil)
}

type Response struct {
	Entities map[string]uint64
}

type Entity struct {
	Value uint64
}

func (e *Entity) Increment() uint64 {
	return atomic.AddUint64(&e.Value, 1)
}

func getSymbols() (symbols []string) {
	r := bufio.NewReader(bytes.NewReader(symbolsData))
	rs := bufio.NewScanner(r)
	var line, symbol string
	for rs.Scan() {
		line = rs.Text()
		symbol, _, _ = strings.Cut(line, "|")
		symbols = append(symbols, symbol)
	}
	return
}
