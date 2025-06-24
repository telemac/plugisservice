package plugisservice

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
)

// ServiceRunner is a struct that runs one or more services.
type ServiceRunner struct {
	wg  sync.WaitGroup
	log *slog.Logger
	nc  *nats.Conn
}

func NewServiceRunner(nc *nats.Conn, log *slog.Logger) *ServiceRunner {
	if log == nil {
		log = slog.Default()
	}
	if nc == nil {
		log.Error("failed to connect to nats")
		return nil
	}
	return &ServiceRunner{
		nc:  nc,
		log: log,
	}
}

// Start starts a service.
func (sr *ServiceRunner) Start(ctx context.Context, svc PlugisServiceIntf) {
	sr.wg.Add(1)
	go func() {
		defer sr.wg.Done()
		svc.SetLogger(sr.log)
		svc.SetNats(sr.nc)
		serviceType := fmt.Sprintf("%T", svc)
		err := svc.Run(ctx)
		if err != nil {
			sr.log.Error("service execution", "error", err, "service", serviceType)
		}
		err = svc.Nats().Flush()
		if err != nil {
			sr.log.Error("service flush", "error", err, "service", fmt.Sprintf("%T", svc))
		}
	}()
}

// Wait waits for all services to finish.
func (sr *ServiceRunner) Wait() {
	sr.wg.Wait()
	if sr.nc != nil {
		err := sr.nc.Drain()
		if err != nil {
			sr.log.Error("service drain", "error", err)
		}
		sr.nc.Close()
	}
}
