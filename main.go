package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	ma "github.com/willianpc/instana-mock-agent/agent"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	port      string
	portsPool int
	portMap   map[int]*ma.Agent
	mainMu    sync.Mutex
)

func init() {
	portsPool = 29090
	port = "9090"

	portMap = make(map[int]*ma.Agent)

	if p := os.Getenv("MOCK_AGENT_PORT"); p != "" {
		port = p
	}
}

func spawnAgent(w http.ResponseWriter, r *http.Request) {
	traceSomething()
	mainMu.Lock()
	defer mainMu.Unlock()

	var newPort int

	p := strings.Split(r.URL.Path, "/")

	if len(p) != 3 {
		// path is neither /spawn/ or /spawn/{port}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// /spawn/ called
	if p[2] == "" {
		portsPool++

		if _, ok := portMap[portsPool]; ok {
			// port already in use
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		newPort = portsPool
	}

	// /spawn/some_port called
	if p[2] != "" {
		agentPort, err := strconv.Atoi(p[2])

		if err != nil {
			// port is not a number
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, ok := portMap[agentPort]; ok {
			// port already in use
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		newPort = agentPort
	}

	if newPort == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	agentSpawn := ma.NewAgent(&ma.Options{
		Addr: ":" + strconv.Itoa(newPort),
	})

	agentSpawn.Start()

	portMap[newPort] = agentSpawn

	w.Header().Add("X-MOCK-AGENT-PORT", strconv.Itoa(newPort))

	_, err := w.Write([]byte(strconv.Itoa(newPort) + "\n"))

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
		err = portMap[agentPort].Stop()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		delete(portMap, agentPort)

		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func agentList(w http.ResponseWriter, r *http.Request) {
	l := ""

	for k := range portMap {
		l += ", " + strconv.Itoa(k)
	}

	_, _ = w.Write([]byte(l))
}

func traceSomething() {
	tracer := otel.Tracer("test-tracer")
	// Attributes represent additional key-value descriptors that can be bound
	// to a metric observer or recorder.
	commonAttrs := []attribute.KeyValue{
		attribute.String("attrA", "chocolate"),
		attribute.String("attrB", "raspberry"),
		attribute.String("attrC", "vanilla"),
	}

	var opts trace.SpanStartOption = trace.WithAttributes(commonAttrs...)

	ctx := context.Background()

	// work begins
	ctx, span := tracer.Start(
		ctx,
		"parent-span",
		opts)

	span.AddEvent("span-created")

	_, childSp := tracer.Start(
		ctx,
		"chid-span",
		trace.WithAttributes(commonAttrs...))

	childSp.RecordError(errors.New("oh nooo!!!!!!"))
	childSp.SetStatus(codes.Error, "oh no")

	span.End()

	childSp.End()
}

func main() {
	if _, ok := os.LookupEnv("ENABLE_JAEGER"); ok {
		ma.EnableJeager()
	}

	http.HandleFunc("/spawn/", spawnAgent)
	http.HandleFunc("/kill/", killAgent)
	http.HandleFunc("/list", agentList)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
