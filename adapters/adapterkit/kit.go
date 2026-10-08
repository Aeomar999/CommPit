package adapterkit

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/core"
)

// Kit encapsulates shared dependencies for provider adapters, offering
// convenience constructors to wrap adapter routes in standard middleware.
type Kit struct {
	resolver       core.ProjectResolver
	bus            core.Bus
	sink           RequestLogSink
	clock          core.Clock
	maxBodySize    int
	defaultProject string
}

// Option configures an adapterkit Kit instance.
type Option func(*Kit)

// WithBus supplies an event bus for publishing request.logged events.
func WithBus(bus core.Bus) Option {
	return func(k *Kit) {
		k.bus = bus
	}
}

// WithSink supplies a sink for persisting RequestLog entries.
func WithSink(sink RequestLogSink) Option {
	return func(k *Kit) {
		k.sink = sink
	}
}

// WithClock supplies a custom clock for deterministic testing.
func WithClock(clock core.Clock) Option {
	return func(k *Kit) {
		k.clock = clock
	}
}

// WithMaxBodySize overrides the default 64 KB payload logging cap.
func WithMaxBodySize(size int) Option {
	return func(k *Kit) {
		k.maxBodySize = size
	}
}

// WithKitDefaultProject configures the fallback project ID for unauthenticated requests.
func WithKitDefaultProject(projectID string) Option {
	return func(k *Kit) {
		k.defaultProject = projectID
	}
}

// New constructs a new Kit with the given project resolver and options.
func New(resolver core.ProjectResolver, opts ...Option) *Kit {
	kit := &Kit{
		resolver:       resolver,
		clock:          core.RealClock{},
		maxBodySize:    DefaultMaxBodySize,
		defaultProject: "default",
	}
	for _, opt := range opts {
		opt(kit)
	}
	return kit
}

// Wrap sets up a Chi router for the given adapter, applying RequestLogger,
// Recoverer, and ResolveProject middleware before delegating to adapter.Routes.
func (k *Kit) Wrap(adapter Adapter, extractor CredentialExtractor, opts ...ResolverOption) http.Handler {
	router := chi.NewRouter()

	router.Use(RequestLogger(adapter, LoggingConfig{
		AdapterName: adapter.Name(),
		Sink:        k.sink,
		Bus:         k.bus,
		Clock:       k.clock,
		MaxBodySize: k.maxBodySize,
	}))

	router.Use(Recoverer(adapter))

	combinedOpts := make([]ResolverOption, 0, len(opts)+1)
	if k.defaultProject != "" {
		combinedOpts = append(combinedOpts, WithDefaultProject(k.defaultProject))
	}
	combinedOpts = append(combinedOpts, opts...)

	router.Use(ResolveProject(adapter, k.resolver, extractor, combinedOpts...))

	adapter.Routes(router)

	return router
}
