package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	port      string
	portsPool int
	portMap   map[int]*agent
	mainMu    sync.Mutex
)

func init() {
	portsPool = 29090
	port = "9090"

	portMap = make(map[int]*agent)

	if p := os.Getenv("MOCK_AGENT_PORT"); p != "" {
		port = p
	}
}

func spawnAgent(w http.ResponseWriter, r *http.Request) {
	mainMu.Lock()
	defer mainMu.Unlock()

	portsPool++

	agentSpawn := &agent{
		port: portsPool,
	}

	agentSpawn.start()

	portMap[portsPool] = agentSpawn

	w.Header().Add("X-MOCK-AGENT-PORT", strconv.Itoa(portsPool))

	_, err := w.Write([]byte(strconv.Itoa(portsPool)))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func killAgent(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(r.URL.Path, "/")

	if len(p) != 3 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	agentPort, err := strconv.Atoi(p[2])

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	mainMu.Lock()
	defer mainMu.Unlock()

	if _, ok := portMap[agentPort]; ok {
		err = portMap[agentPort].stop()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		delete(portMap, agentPort)
		fmt.Println(portMap)

		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func main() {
	http.HandleFunc("/spawn", spawnAgent)
	http.HandleFunc("/kill/", killAgent)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
