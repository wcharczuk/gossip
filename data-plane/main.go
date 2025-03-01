package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"gossip/pkg/types"
	"net/http"
	"os"
	"strings"
	"time"

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
		entities[symbol] = &Entity{LastSeen: time.Now()}
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
		response := types.DataPlaneResponse{
			Entities: make(map[string]int64),
		}
		if len(requestSymbols) > 0 {
			for _, requestSymbol := range requestSymbols {
				response.Entities[requestSymbol] = int64(entities[requestSymbol].Increment())
			}
		} else {
			for symbol, entity := range entities {
				response.Entities[symbol] = int64(entity.Increment())
			}
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(response)
	}))
	_ = http.ListenAndServe(bindAddr(), nil)
}

type Entity struct {
	LastSeen time.Time
}

func (e *Entity) Increment() time.Duration {
	now := time.Now()
	elapsed := now.Sub(e.LastSeen)
	e.LastSeen = time.Now()
	return elapsed
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
