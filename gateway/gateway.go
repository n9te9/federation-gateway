package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/n9te9/go-graphql-federation-gateway/federation/executor"
	"github.com/n9te9/go-graphql-federation-gateway/federation/graph"
	"github.com/n9te9/go-graphql-federation-gateway/federation/planner"
	"github.com/n9te9/goliteql/query"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type GatewayService struct {
	Name        string   `yaml:"name"`
	Host        string   `yaml:"host"`
	SchemaFiles []string `yaml:"schema_files"`
}

type GatewayOption struct {
	Endpoint                    string               `yaml:"endpoint"`
	ServiceName                 string               `yaml:"service_name"`
	Port                        int                  `yaml:"port"`
	TimeoutDuration             string               `yaml:"timeout_duration" default:"5s"`
	EnableHangOverRequestHeader bool                 `yaml:"enable_hang_over_request_header" default:"true"`
	Services                    []GatewayService     `yaml:"services"`
	Opentelemetry               OpentelemetrySetting `yaml:"opentelemetry"`
}

type OpentelemetrySetting struct {
	TracingSetting OpentelemetryTracingSetting `yaml:"tracing"`
}

type OpentelemetryTracingSetting struct {
	Enable bool `yaml:"enable" default:"false"`
}

type gateway struct {
	graphQLEndpoint string
	serviceName     string
	planner         planner.Planner
	executor        executor.Executor
	superGraph      *graph.SuperGraph
	queryParser     *query.Parser

	enableComplementRequestId   bool
	enableHangOverRequestHeader bool
	enableOpentelemetryTracing  bool
}

var _ http.Handler = (*gateway)(nil)

func readSchemaFiles(paths []string) ([]byte, error) {
	ret := make([]byte, 0)
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file(%s): %w", path, err)
		}

		ret = append(ret, b...)
		ret = append(ret, '\n')
	}

	return ret, nil
}

func NewGateway(settings *GatewayOption) (*gateway, error) {
	subGraphs := make([]*graph.SubGraph, 0, len(settings.Services))
	allSchemaSrc := []byte{}

	for _, srv := range settings.Services {
		schema, err := readSchemaFiles(srv.SchemaFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema file: %w", err)
		}

		subGraph, err := graph.NewSubGraph(srv.Name, schema, srv.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to create subgraph for service %s: %w", srv.Name, err)
		}
		subGraphs = append(subGraphs, subGraph)
		allSchemaSrc = append(allSchemaSrc, schema...)
	}

	superGraph, err := graph.NewSuperGraph(allSchemaSrc, subGraphs)
	if err != nil {
		return nil, fmt.Errorf("failed to create supergraph: %w", err)
	}

	plannerOption := planner.PlannerOption{
		EnableOpentelemetryTracing: settings.Opentelemetry.TracingSetting.Enable,
	}

	executorOption := executor.ExecutorOption{
		EnableOpentelemetryTracing: settings.Opentelemetry.TracingSetting.Enable,
	}

	httpClient := &http.Client{}
	if settings.Opentelemetry.TracingSetting.Enable {
		httpClient.Transport = otelhttp.NewTransport(&http.Transport{})
	}

	return &gateway{
		graphQLEndpoint:             settings.Endpoint,
		superGraph:                  superGraph,
		planner:                     planner.NewPlanner(superGraph, plannerOption),
		enableHangOverRequestHeader: settings.EnableHangOverRequestHeader,
		serviceName:                 settings.ServiceName,
		executor:                    executor.NewExecutor(httpClient, superGraph, executorOption),
		queryParser:                 query.NewParserWithLexer(),
		enableOpentelemetryTracing:  settings.Opentelemetry.TracingSetting.Enable,
	}, nil
}

func (g *gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case g.graphQLEndpoint:
		g.Routing(w, r)
	}
}

type Request struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

func (g *gateway) Routing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{
					"message": err.Error(),
				},
			},
		})
		return
	}

	document, err := g.queryParser.Parse([]byte(req.Query))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{
					"message": err.Error(),
				},
			},
		})
		return
	}

	plan, err := g.planner.Plan(document, req.Variables)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{
					"message": err.Error(),
				},
			},
		})
		return
	}

	ctx := r.Context()
	header := http.Header{}
	if g.enableComplementRequestId {
		requestId := r.Header.Get("X-Request-Id")
		if requestId == "" {
			requestId = uuid.NewString()
		}

		header.Set("X-Request-Id", requestId)
		r.Header.Set("X-Request-Id", requestId)
		ctx = executor.SetRequestHeaderToContext(ctx, header)
	}

	if g.enableHangOverRequestHeader {
		ctx = executor.SetRequestHeaderToContext(ctx, r.Header)
	} else {
		ctx = executor.SetRequestHeaderToContext(ctx, header)
	}

	resp := g.executor.Execute(ctx, plan, req.Variables)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
