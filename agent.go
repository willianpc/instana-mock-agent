package main

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
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

type agent struct {
	port int
	*http.Server
	dumpedSpans    []span
	endpointCombos []urlMatch
	mu             *sync.Mutex
}

func (a *agent) initServer() {
	if a.Server == nil {
		a.mu = &sync.Mutex{}
		mux := http.NewServeMux()

		a.endpointCombos = []urlMatch{
			{
				tracesRE,
				func(w http.ResponseWriter, r *http.Request) {
					tracesHandler(w, r, func(spans []span) {
						a.mu.Lock()
						a.dumpedSpans = append(a.dumpedSpans, spans...)
						a.mu.Unlock()
					})
				},
			},
			{
				discoveryRE,
				discoveryHandler,
			},
			{
				pingRE,
				pingHandler,
			},
		}

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

		a.Server = &http.Server{
			Addr:    ":" + strconv.Itoa(a.port),
			Handler: mux,
		}
	}
}

func (a *agent) start() {
	a.initServer()

	go func() {
		_ = a.Server.ListenAndServe()
	}()
}

func (a *agent) stop() error {
	if a.Server != nil {
		return a.Server.Shutdown(context.Background())
	}

	return nil
}
