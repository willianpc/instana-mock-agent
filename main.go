package main

import (
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
	endpointCombos []urlMatch
	dumpedSpans    []span
	mu             sync.Mutex
)

func init() {
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
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for _, endpointCombo := range endpointCombos {
			if endpointCombo.re.MatchString(r.URL.String()) {
				endpointCombo.h(w, r)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	})

	http.HandleFunc("/dump", dumpHandler)

	log.Fatal(http.ListenAndServe(":9090", nil))
}
