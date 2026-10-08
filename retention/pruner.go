package retention

import (
	"context"
	"log/slog"
	"time"

	"github.com/Aeomar999/CommPit/config"
	"github.com/Aeomar999/CommPit/core"
)

type Pruner struct {
	cfg     *config.RetentionConfig
	store   core.Store
	service *core.Service
	ticker  *time.Ticker
	done    chan struct{}
}

func NewPruner(cfg *config.RetentionConfig, store core.Store, service *core.Service) *Pruner {
	return &Pruner{
		cfg:     cfg,
		store:   store,
		service: service,
		done:    make(chan struct{}),
	}
}

func (p *Pruner) Start(ctx context.Context) {
	if !p.cfg.Enabled {
		slog.Info("Retention pruner disabled")
		return
	}

	p.ticker = time.NewTicker(p.cfg.Interval)
	slog.Info("Retention pruner started", "interval", p.cfg.Interval)

	go func() {
		for {
			select {
			case <-p.ticker.C:
				p.prune(ctx)
			case <-p.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (p *Pruner) Stop() {
	if p.ticker != nil {
		p.ticker.Stop()
	}
	close(p.done)
}

func (p *Pruner) prune(ctx context.Context) {
	slog.Debug("Starting retention prune")

	// Prune messages older than MessageTTL
	if p.cfg.MessageTTL > 0 {
		deleted, err := p.store.DeleteMessagesOlderThan(ctx, p.cfg.MessageTTL)
		if err != nil {
			slog.Error("Failed to prune messages", "error", err)
		} else if deleted > 0 {
			slog.Info("Pruned messages", "count", deleted)
		}
	}

	// Prune verifications older than VerificationTTL
	if p.cfg.VerificationTTL > 0 {
		deleted, err := p.store.DeleteVerificationsOlderThan(ctx, p.cfg.VerificationTTL)
		if err != nil {
			slog.Error("Failed to prune verifications", "error", err)
		} else if deleted > 0 {
			slog.Info("Pruned verifications", "count", deleted)
		}
	}

	// Prune batches older than BatchTTL
	if p.cfg.BatchTTL > 0 {
		deleted, err := p.store.DeleteBatchesOlderThan(ctx, p.cfg.BatchTTL)
		if err != nil {
			slog.Error("Failed to prune batches", "error", err)
		} else if deleted > 0 {
			slog.Info("Pruned batches", "count", deleted)
		}
	}

	// Prune request logs older than RequestLogTTL
	if p.cfg.RequestLogTTL > 0 {
		deleted, err := p.store.DeleteRequestLogsOlderThan(ctx, p.cfg.RequestLogTTL)
		if err != nil {
			slog.Error("Failed to prune request logs", "error", err)
		} else if deleted > 0 {
			slog.Info("Pruned request logs", "count", deleted)
		}
	}

	// Prune webhook deliveries older than WebhookTTL
	if p.cfg.WebhookTTL > 0 {
		deleted, err := p.store.DeleteWebhookDeliveriesOlderThan(ctx, p.cfg.WebhookTTL)
		if err != nil {
			slog.Error("Failed to prune webhook deliveries", "error", err)
		} else if deleted > 0 {
			slog.Info("Pruned webhook deliveries", "count", deleted)
		}
	}

	slog.Debug("Retention prune completed")
}
