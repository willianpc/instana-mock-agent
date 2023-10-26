package agent

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"sync"
)

type urlMatch struct {
	re regexp.Regexp
	h  http.HandlerFunc
}

var (
	tracesRE, discoveryRE, pingRE regexp.Regexp
)

func init() {
	// eg: /com.instana.plugin.golang/traces.12345
	tracesRE = *regexp.MustCompile(`\/com\.instana\.plugin\..*\/traces\.\d+`)

	// eg: /com.instana.plugin.golang.discovery
	discoveryRE = *regexp.MustCompile(`\/com\.instana\.plugin\..*\.discovery`)

	// eg: /com.instana.plugin.golang.12345
	pingRE = *regexp.MustCompile(`\/com\.instana\.plugin\..*\.\d+`)
}

type Options struct {
	Addr    string
	PID     uint32
	HostID  string
	Secrets struct {
		Matcher string
		List    []string
	}
	ExtraHTTPHeaders []string
	Tracing          struct {
		ExtraHTTPHeaders []string
	}
}

type Agent struct {
	*http.Server
	dumpedSpans    []span
	endpointCombos []urlMatch
	mu             *sync.Mutex
}

func NewAgent(opts *Options) *Agent {
	if opts == nil {
		log.Println("opts not provided")
		return nil
	}

	if opts.Addr == "" {
		log.Println("opts.Addr not provided")
		return nil
	}

	a := &Agent{
		Server: &http.Server{
			Addr: opts.Addr,
		},
	}

	a.initServer()

	return a
}

func (a *Agent) initServer() {
	if a.Server == nil || a.Server.Handler == nil || a.Server.Addr == "" {
		a.mu = &sync.Mutex{}
		mux := http.NewServeMux()

		a.initEndpointCombos()

		mux.HandleFunc("/dump", func(w http.ResponseWriter, r *http.Request) {
			dumpHandler(w, r, func() []span {
				a.mu.Lock()
				defer a.mu.Unlock()
				return a.dumpedSpans
			})
		})

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			urlPath := r.URL.String()

			if urlPath == "/" {
				w.WriteHeader(http.StatusOK)
				return
			}

			for _, endpointCombo := range a.endpointCombos {
				if endpointCombo.re.MatchString(urlPath) {
					endpointCombo.h(w, r)
					return
				}
			}

			w.WriteHeader(http.StatusNotFound)
		})

		a.Server.Handler = mux
	}
}

func (a *Agent) initEndpointCombos() {
	a.endpointCombos = []urlMatch{
		{
			tracesRE,
			a.traceHandler,
		},
		{
			discoveryRE,
			a.discoveryHandler,
		},
		{
			pingRE,
			pingHandler,
		},
	}
}

func (a *Agent) traceHandler(w http.ResponseWriter, r *http.Request) {
	tracesHandler(w, r, func(spans []span) {
		a.mu.Lock()
		a.dumpedSpans = append(a.dumpedSpans, spans...)
		a.mu.Unlock()
	})
}

func (a *Agent) discoveryHandler(w http.ResponseWriter, r *http.Request) {
	discoveryHandler(w, r, func(dr discoveryRequest) discoveryResponse {
		pid := dr.PID

		if pid == 0 {
			pid = 1
		}

		return discoveryResponse{PID: pid, HostID: "88:66:5a:ff:fe:05:a5:f0"}
	})
}

func (a *Agent) Start() {
	a.initServer()

	go func() {
		_ = a.Server.ListenAndServe()
	}()
}

func (a *Agent) Stop() error {
	if a.Server != nil {
		return a.Server.Shutdown(context.Background())
	}

	return nil
}
