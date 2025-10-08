package gql

import (
	"fmt"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/prometheus/client_golang/prometheus"
	prometheushttp "github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/zap"

	"github.com/stormhead-org/stormic-backend/iam/internal/graph"
	metricpkg "github.com/stormhead-org/stormic-backend/iam/internal/metric"
)

type GQL struct {
	logger   *zap.Logger
	host     string
	port     string
	registry *prometheus.Registry
}

func NewGQL(logger *zap.Logger, host string, port string) *GQL {
	gqlServer := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

	gqlServer.AddTransport(transport.Options{})
	gqlServer.AddTransport(transport.GET{})
	gqlServer.AddTransport(transport.POST{})

	gqlServer.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	gqlServer.Use(extension.Introspection{})
	gqlServer.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	this := &GQL{
		logger:   logger,
		host:     host,
		port:     port,
		registry: metricpkg.CreateRegistry(),
	}

	// GQL handlers
	if os.Getenv("DEBUG") == "1" {
		http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	}
	http.Handle("/query", gqlServer)

	// Prometheus metrics
	http.Handle("/metrics", prometheushttp.HandlerFor(this.registry, prometheushttp.HandlerOpts{Registry: this.registry}))

	// Kubernetes probes
	http.Handle("/probe/startup", this.startupHandler())
	http.Handle("/probe/readiness", this.readinessHandler())
	http.Handle("/probe/liveness", this.livenessHandler())

	return this
}

func (this *GQL) Start() error {
	this.logger.Info("start")

	go func() {
		http.ListenAndServe(
			fmt.Sprintf("%s:%s", this.host, this.port),
			nil,
		)
	}()

	return nil
}

func (this *GQL) Stop() error {
	this.logger.Info("stop")
	return nil
}

func (this *GQL) startupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func (this *GQL) readinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func (this *GQL) livenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}
