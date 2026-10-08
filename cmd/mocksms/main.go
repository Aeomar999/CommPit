package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Aeomar999/CommPit/adapters/adapterkit"
	"github.com/Aeomar999/CommPit/adapters/termii"
	"github.com/Aeomar999/CommPit/adapters/twilio"
	"github.com/Aeomar999/CommPit/api"
	"github.com/Aeomar999/CommPit/bus"
	"github.com/Aeomar999/CommPit/config"
	"github.com/Aeomar999/CommPit/core"
	"github.com/Aeomar999/CommPit/middleware"
	"github.com/Aeomar999/CommPit/phone"
	"github.com/Aeomar999/CommPit/retention"
	"github.com/Aeomar999/CommPit/sim"
	"github.com/Aeomar999/CommPit/smtpd"
	"github.com/Aeomar999/CommPit/store/sqlite"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cfg := config.Load()

	flag.StringVar(&cfg.HTTP.Host, "host", cfg.HTTP.Host, "HTTP server host")
	flag.IntVar(&cfg.HTTP.Port, "port", cfg.HTTP.Port, "HTTP server port")
	flag.StringVar(&cfg.SMTP.Host, "smtp-host", cfg.SMTP.Host, "SMTP server host")
	flag.IntVar(&cfg.SMTP.Port, "smtp-port", cfg.SMTP.Port, "SMTP server port")
	flag.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "Data directory")
	flag.BoolVar(&cfg.Memory, "memory", cfg.Memory, "Use in-memory SQLite")
	flag.DurationVar(&cfg.Lifecycle.StepDelay, "step-delay", cfg.Lifecycle.StepDelay, "Lifecycle step delay")
	flag.StringVar(&cfg.OTP.FixedCode, "otp-code", cfg.OTP.FixedCode, "Fixed OTP code for testing")
	flag.StringVar(&cfg.Validation.Phone, "phone-validation", cfg.Validation.Phone, "Phone validation mode: valid, possible, off")
	flag.StringVar(&cfg.UIAuth, "ui-auth", cfg.UIAuth, "UI basic auth user:pass")
	flag.BoolVar(&cfg.NoDockerHostRewrite, "no-docker-host-rewrite", cfg.NoDockerHostRewrite, "Disable Docker host rewrite")
	flag.BoolVar(&cfg.Version, "version", false, "Print version and exit")
	flag.Parse()

	if cfg.Version {
		fmt.Printf("mocksms %s (commit: %s, date: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return runWithContext(ctx, cfg)
}

func runWithContext(ctx context.Context, cfg *config.Config) error {
	var dataDir string
	if cfg.Memory {
		dataDir = ":memory:"
	} else {
		dataDir = cfg.DataDir
	}

	store, err := sqlite.NewStore(dataDir, cfg.Store.ReadPoolSize)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	defer store.Close()

	blobStore := store

	eventBus := bus.NewEventBus()

	simulator := sim.NewSimulator()
	simulator.SetLatency(cfg.Sim.Latency)
	simulator.SetFailureRate(cfg.Sim.FailureRate)

	var phoneMode phone.Mode
	switch cfg.Validation.Phone {
	case "possible":
		phoneMode = phone.ModePossible
	case "off":
		phoneMode = phone.ModeOff
	default:
		phoneMode = phone.ModeValid
	}

	var otpFixedCode *string
	if cfg.OTP.FixedCode != "" {
		otpFixedCode = &cfg.OTP.FixedCode
	}

	projectResolver := core.NewProjectResolver(store)

	service := core.NewService(core.ServiceConfig{
		Store:        store,
		BlobStore:    blobStore,
		Bus:          eventBus,
		Simulator:    simulator,
		Clock:        core.RealClock{},
		Resolver:     projectResolver,
		StepDelay:    cfg.Lifecycle.StepDelay,
		OTPFixedCode: otpFixedCode,
		PhoneMode:    phoneMode,
	})

	handlers := api.NewHandlers(service, projectResolver, eventBus, version, &cfg.Security)

	service.LifecycleRunner().ResumeQueuedAndSent(ctx)

	// Start SMTP server
	smtpServer := smtpd.NewServer(&smtpd.Config{
		Host:           cfg.SMTP.Host,
		Port:           cfg.SMTP.Port,
		MaxMessageSize: 25 * 1024 * 1024, // 25 MB
		EnableSTARTTLS: false,
	}, service, projectResolver)

	smtpErrCh := make(chan error, 1)
	go func() {
		fmt.Printf("Starting SMTP server on %s:%d\n", cfg.SMTP.Host, cfg.SMTP.Port)
		if err := smtpServer.Start(); err != nil {
			smtpErrCh <- err
		}
	}()

	// Start retention pruner
	pruner := retention.NewPruner(&cfg.Retention, store, service)
	pruner.Start(ctx)

	// Twilio adapter: provider-format routes under /twilio with request
	// logging persisted to the store. Credentials are required, matching
	// the real API (401 without them).
	twilioAdapter := twilio.New(service)
	twilioSink := adapterkit.RequestLogSinkFunc(func(ctx context.Context, entry *core.RequestLog) error {
		return store.CreateRequestLog(ctx, entry)
	})
	twilioKit := adapterkit.New(projectResolver, adapterkit.WithBus(eventBus), adapterkit.WithSink(twilioSink))
	twilioHandler := twilioKit.Wrap(twilioAdapter, twilio.Extractor(), adapterkit.WithRequired(true))

	// Termii adapter: same middleware stack, api_key credential from the
	// JSON body (bulk bodies need a larger extraction cap).
	termiiAdapter := termii.New(service)
	termiiSink := adapterkit.RequestLogSinkFunc(func(ctx context.Context, entry *core.RequestLog) error {
		return store.CreateRequestLog(ctx, entry)
	})
	termiiKit := adapterkit.New(projectResolver, adapterkit.WithBus(eventBus), adapterkit.WithSink(termiiSink))
	termiiHandler := termiiKit.Wrap(termiiAdapter, termii.Extractor(), adapterkit.WithRequired(true))

	rootRouter := chi.NewRouter()
	rootRouter.Mount("/twilio", middleware.SecurityMiddleware(&cfg.Security)(twilioHandler))
	rootRouter.Mount("/termii", middleware.SecurityMiddleware(&cfg.Security)(termiiHandler))
	rootRouter.Mount("/", handlers.Routes())

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      rootRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		fmt.Printf("Starting HTTP server on %s:%d\n", cfg.HTTP.Host, cfg.HTTP.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		fmt.Println("Shutting down...")
	case err := <-serverErrCh:
		runErr = fmt.Errorf("HTTP server error: %w", err)
	case err := <-smtpErrCh:
		runErr = fmt.Errorf("SMTP server error: %w", err)
	}

	// 1. Drain HTTP server (architecture.md §7)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "Server shutdown error: %v\n", err)
	}

	// 2. Stop SMTP server
	if err := smtpServer.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "SMTP shutdown error: %v\n", err)
	}

	// 3. Stop retention pruner
	pruner.Stop()

	// 4. Stop lifecycle runner
	service.Shutdown()

	// 5. store.Close() is called by defer

	return runErr
}
