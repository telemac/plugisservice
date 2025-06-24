package nats_service

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

// EndpointConfig defines configuration for a single NATS endpoint
type EndpointConfig struct {
	Name           string               `json:"name"`
	Handler        PlugisServiceHandler `json:"-"` // Non-serializable
	MaxConcurrency int                  `json:"max_concurrency"`
	RequestTimeout time.Duration        `json:"request_timeout"`
	Metadata       map[string]string    `json:"metadata,omitempty"`
	QueueGroup     string               `json:"queue_group,omitempty"`
	Subject        string               `json:"subject,omitempty"`
}

// setDefaults applies default values to endpoint configuration
func (config *EndpointConfig) setDefaults(serviceName, prefix string) {
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 10
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 30 * time.Second
	}
	if config.Subject == "" {
		if prefix == "" {
			config.Subject = serviceName + "." + config.Name
		} else {
			config.Subject = prefix + "." + serviceName + "." + config.Name
		}
	}
}

// NatsService wraps a NATS micro.Service with endpoint management
type NatsService struct {
	nc     *nats.Conn
	prefix string
	svc    micro.Service
}

// NewNatsService creates a new NATS service with the given prefix and configuration
func NewNatsService(conn *nats.Conn, prefix string, config micro.Config) (*NatsService, error) {
	natsService := &NatsService{
		nc:     conn,
		prefix: prefix,
	}
	var err error
	natsService.svc, err = micro.AddService(natsService.nc, config)
	return natsService, err
}

// Svc returns the underlying NATS micro.Service
func (ns *NatsService) Svc() micro.Service {
	return ns.svc
}

// Stop stops the NATS service
func (ns *NatsService) Stop() error {
	return ns.svc.Stop()
}

// Shutdown gracefully shuts down the service with context cancellation
func (ns *NatsService) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ns.svc.Stop()
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PlugisServiceReply defines the standard response format for all endpoints
type PlugisServiceReply struct {
	Error     string    `json:"error,omitempty"`
	Result    any       `json:"result,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Duration  string    `json:"duration,omitempty"`
}

// PlugisServiceHandler is the function signature for endpoint handlers
type PlugisServiceHandler func(ctx context.Context, request micro.Request) (any, error)

// PlugisHandler manages concurrency and timeouts for a single endpoint
type PlugisHandler struct {
	plugisServiceHandler PlugisServiceHandler
	ctx                  context.Context
	semaphore            chan struct{}
	config               EndpointConfig
}

// NewPlugisHandler creates a new handler with concurrency control
func NewPlugisHandler(ctx context.Context, config EndpointConfig) *PlugisHandler {
	return &PlugisHandler{
		ctx:                  ctx,
		plugisServiceHandler: config.Handler,
		semaphore:            make(chan struct{}, config.MaxConcurrency),
		config:               config,
	}
}

// Handle processes incoming requests with concurrency limiting
func (ph PlugisHandler) Handle(req micro.Request) {
	// Try to acquire semaphore with non-blocking select
	select {
	case ph.semaphore <- struct{}{}:
		// Successfully acquired semaphore, process request
		go ph.handleWithTimeout(req)
	default:
		// Endpoint overloaded, reject request immediately
		ph.sendErrorResponse(req, "503", "Endpoint overloaded - too many concurrent requests", time.Now())
	}
}

// handleWithTimeout executes the handler with timeout and panic recovery
func (ph PlugisHandler) handleWithTimeout(req micro.Request) {
	start := time.Now()

	// Always release semaphore when done
	defer func() {
		<-ph.semaphore
	}()

	// Create timeout context
	ctx, cancel := context.WithTimeout(ph.ctx, ph.config.RequestTimeout)
	defer cancel()

	// Channel to receive result
	type result struct {
		data any
		err  error
	}
	resultChan := make(chan result, 1)

	// Execute handler in separate goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- result{
					data: nil,
					err:  fmt.Errorf("handler panicked: %v", r),
				}
			}
		}()

		data, err := ph.plugisServiceHandler(ctx, req)
		resultChan <- result{data: data, err: err}
	}()

	// Wait for either completion or timeout
	select {
	case res := <-resultChan:
		ph.sendResponse(req, res.data, res.err, start)
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			ph.sendErrorResponse(req, "408", "Request timeout", start)
		} else {
			ph.sendErrorResponse(req, "499", "Request cancelled", start)
		}
	}
}

// sendResponse sends a successful response with timing information
func (ph PlugisHandler) sendResponse(req micro.Request, result any, err error, start time.Time) {
	duration := time.Since(start)

	reply := PlugisServiceReply{
		Result:    result,
		Timestamp: time.Now(),
		Duration:  duration.String(),
	}

	if err != nil {
		reply.Error = err.Error()
	}

	if err := req.RespondJSON(reply); err != nil {
		// Failed to send response, but we can't do much about it
		// In a production system, this would be logged
	}
}

// sendErrorResponse sends an error response with status code and timing
func (ph PlugisHandler) sendErrorResponse(req micro.Request, code, message string, start time.Time) {
	duration := time.Since(start)

	reply := PlugisServiceReply{
		Error:     fmt.Sprintf("%s: %s", code, message),
		Timestamp: time.Now(),
		Duration:  duration.String(),
	}

	if err := req.RespondJSON(reply); err != nil {
		// Failed to send error response
	}
}

// AddEndpoint adds a new endpoint to the service with the given configuration
func (ns *NatsService) AddEndpoint(ctx context.Context, config EndpointConfig) error {
	if config.Name == "" {
		return fmt.Errorf("endpoint name cannot be empty")
	}
	if config.Handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	// Set defaults
	config.setDefaults(ns.svc.Info().Name, ns.prefix)

	// Build micro options
	options := []micro.EndpointOpt{
		micro.WithEndpointSubject(config.Subject),
	}

	if config.QueueGroup != "" {
		options = append(options, micro.WithEndpointQueueGroup(config.QueueGroup))
	} else {
		options = append(options, micro.WithEndpointQueueGroupDisabled())
	}

	if len(config.Metadata) > 0 {
		options = append(options, micro.WithEndpointMetadata(config.Metadata))
	}

	handler := NewPlugisHandler(ctx, config)

	return ns.svc.AddEndpoint(config.Name, handler, options...)
}

func (ns *NatsService) Prefix() string {
	return ns.prefix
}
