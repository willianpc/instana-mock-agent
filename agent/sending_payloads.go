package agent

type discoveryResponse struct {
	Pid     uint32 `json:"pid"`
	HostID  string `json:"agentUuid"`
	Secrets struct {
		Matcher string   `json:"matcher"`
		List    []string `json:"list"`
	} `json:"secrets"`
	ExtraHTTPHeaders []string `json:"extraHeaders"`
	Tracing          struct {
		ExtraHTTPHeaders []string `json:"extra-http-headers"`
	} `json:"tracing"`
}
