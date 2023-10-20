package main

import (
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
)

type urlMatch struct {
	re regexp.Regexp
	h  http.HandlerFunc
}

var (
	endpointCombos []urlMatch
	dumpedSpans    []span
	mu             sync.Mutex
	port           string
)

func init() {
	// official port is 42699
	port = "42698"

	if p := os.Getenv("MOCK_AGENT_PORT"); p != "" {
		port = p
	}

	endpointCombos = []urlMatch{
		{
			// eg: /com.instana.plugin.golang/traces.12345
			*regexp.MustCompile(`\/com\.instana\.plugin\..*\/traces\.\d+`),
			spanHandler,
		},
		{
			// eg: /com.instana.plugin.golang.discovery
			*regexp.MustCompile(`\/com\.instana\.plugin\..*\.discovery`),
			discoveryHandler,
		},
		{
			// eg: /com.instana.plugin.golang.12345
			*regexp.MustCompile(`\/com\.instana\.plugin\..*\.\d+`),
			pingHandler,
		},
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.String()

		if urlPath == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}

		for _, endpointCombo := range endpointCombos {
			if endpointCombo.re.MatchString(urlPath) {
				endpointCombo.h(w, r)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	})

	http.HandleFunc("/dump", dumpHandler)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
