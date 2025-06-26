package plugisservice

import (
	"context"
	nats_service "github.com/telemac/plugisservice/pkg/nats-service"
	"iter"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/synadia-io/orbit.go/natsext"
)

// Metadata is a map of key-value pairs that can be used to store additional information about a service/endpoint
type Metadata map[string]string

// PlugisServiceIntf is the interface that must be implemented by all plugis services.
type PlugisServiceIntf interface {
	Run(ctx context.Context) error
	Name() string
	Description() string
	Version() string
	Metadata() Metadata
	PlugisIntf
}

// PlugisIntf holds the methods that can be used by a plugis service.
type PlugisIntf interface {
	SetLogger(*slog.Logger)
	Logger() *slog.Logger
	SetNats(*nats.Conn)
	Nats() *nats.Conn
	SetPrefix(prefix string)
	Prefix() string
	Publish(topic string, payload []byte) error
	Request(subj string, data []byte, timeout time.Duration) (*nats.Msg, error)
	RequestMany(ctx context.Context, subject string, data []byte, opts ...natsext.RequestManyOpt) (iter.Seq2[*nats.Msg, error], error)
	GetServices(ctx context.Context) ([]ServiceInfo, error)
	StartService(svc PlugisServiceIntf) (*nats_service.NatsService, error)
}

// ServiceInfo is the information about a service.
type ServiceInfo struct {
	Name        string     `json:"name"`
	Id          string     `json:"id"`
	Description string     `json:"description"`
	Version     string     `json:"version"`
	Type        string     `json:"type"`
	Metadata    Metadata   `json:"metadata"` // contains at least ServiceMetadata fields
	Endpoints   []Endpoint `json:"endpoints"`
}

// ServiceMetadata is the metadata about a service.
type ServiceMetadata struct {
	Hostname  string `json:"hostname"`
	MAC       string `json:"mac"`
	Platform  string `json:"platform"`
	Prefix    string `json:"prefix"`
	StartedAt string `json:"started_at"`
}

// Endpoint is the information about an endpoint.
type Endpoint struct {
	Name       string   `json:"name"`
	Subject    string   `json:"subject"`
	QueueGroup string   `json:"queue_group"`
	Metadata   Metadata `json:"metadata,omitempty"`
}
