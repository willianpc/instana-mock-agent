package agent

type span struct {
	traceReference

	SpanID          string          `json:"s"`
	LongTraceID     string          `json:"lt,omitempty"`
	Timestamp       uint64          `json:"ts"`
	Duration        uint64          `json:"d"`
	Name            string          `json:"n"`
	From            *fromS          `json:"f"`
	Batch           *batchInfo      `json:"b,omitempty"`
	Kind            int             `json:"k"`
	Ec              int             `json:"ec,omitempty"`
	Data            typedSpanData   `json:"data"`
	Synthetic       bool            `json:"sy,omitempty"`
	CorrelationType string          `json:"crtp,omitempty"`
	CorrelationID   string          `json:"crid,omitempty"`
	ForeignTrace    bool            `json:"tp,omitempty"`
	Ancestor        *traceReference `json:"ia,omitempty"`
}

type traceReference struct {
	TraceID  string `json:"t"`
	ParentID string `json:"p,omitempty"`
}

type fromS struct {
	// By spec, this must be a number, but some tracers send strings, and the Agent accept them.
	EntityID interface{} `json:"e"`
	// Serverless agents fields
	Hostless      bool   `json:"hl,omitempty"`
	CloudProvider string `json:"cp,omitempty"`
	// Host agent fields
	HostID string `json:"h,omitempty"`
}

type batchInfo struct {
	Size int `json:"s"`
}

// represents span.data
type typedSpanData interface{}

type discoveryRequest struct {
	PID               uint32   `json:"pid"`
	Name              string   `json:"name"`
	Args              []string `json:"args"`
	Fd                string   `json:"fd"`
	Inode             string   `json:"inode"`
	CPUSetFileContent string   `json:"cpuSetFileContent,omitempty"`
}
