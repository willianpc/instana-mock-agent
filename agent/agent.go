package agent

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
		mu: &sync.Mutex{},
		Server: &http.Server{
			Addr: opts.Addr,
		},
	}

	a.initServer()

	return a
}

func (a *Agent) initServer() {
	if a.Server == nil || a.Server.Handler == nil || a.Server.Addr == "" {
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
		a.sendSpanToJaeger(spans)
		a.mu.Lock()
		a.dumpedSpans = append(a.dumpedSpans, spans...)
		a.mu.Unlock()
	})
}

func (a *Agent) sendSpanToJaeger(spans []span) {
	fmt.Println(spans)
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

// func newTraceProvider(res *resource.Resource) (*trace.TracerProvider, error) {
// 	traceExporter, err := stdouttrace.New(
// 		stdouttrace.WithPrettyPrint())
// 	if err != nil {
// 		return nil, err
// 	}

// 	traceProvider := trace.NewTracerProvider(
// 		trace.WithBatcher(traceExporter,
// 			// Default is 5s. Set to 1s for demonstrative purposes.
// 			trace.WithBatchTimeout(time.Second)),
// 		trace.WithResource(res),
// 	)
// 	return traceProvider, nil
// }

func initProvider() (func(context.Context) error, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceName("test-service"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of your cluster. If you run the app inside k8s, then you can
	// probably connect directly to the service through dns.
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "localhost:4317",
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

func Lala() {
	log.Printf("Waiting for connection...")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shutdown, err := initProvider()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal("failed to shutdown TracerProvider: %w", err)
		}
	}()

	tracer := otel.Tracer("test-tracer")

	// Attributes represent additional key-value descriptors that can be bound
	// to a metric observer or recorder.
	commonAttrs := []attribute.KeyValue{
		attribute.String("attrA", "chocolate"),
		attribute.String("attrB", "raspberry"),
		attribute.String("attrC", "vanilla"),
	}

	// work begins
	ctx, span := tracer.Start(
		ctx,
		"CollectorExporter-Example",
		trace.WithAttributes(commonAttrs...))
	defer span.End()
	log.Println("veio aki")
}
