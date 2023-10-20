package main

import (
	"log"
	"net/http"
	"regexp"
)

type urlMatch struct {
	re regexp.Regexp
	h  http.HandlerFunc
}

var (
	endpointCombos []urlMatch
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

	log.Fatal(http.ListenAndServe(":9090", nil))
}
