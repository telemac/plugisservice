package plugisservice

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/nats-io/nats.go/micro"
	"github.com/telemac/plugisservice/model"
	nats_service "github.com/telemac/plugisservice/pkg/nats-service"

	"github.com/telemac/goutils/net"
	"github.com/telemac/goutils/task"

	"github.com/nats-io/nats.go"
	"github.com/synadia-io/orbit.go/natsext"
)

// Ensure Plugis implements PlugisIntf
var _ PlugisIntf = (*Plugis)(nil)

// Plugis is the default implementation of the PlugisIntf that provides
// the functionality to plugis services.
type Plugis struct {
	logger *slog.Logger
	nc     *nats.Conn
	prefix string
}

var (
	// ErrNatsNotConnected  = errors.New("nats not connected")
	ErrNatsConnectionNil = errors.New("nats connection is nil")
)

type Event[T any] struct {
	Type string `json:"type,omitempty"`
	Data T      `json:"data,omitempty"`
}

// SetLogger sets the logger for the Plugis instance.
func (plugis *Plugis) SetLogger(log *slog.Logger) {
	plugis.logger = log
}

// Logger returns the logger for the Plugis instance.
func (plugis *Plugis) Logger() *slog.Logger {
	if plugis.logger == nil {
		plugis.logger = slog.Default()
	}
	return plugis.logger
}

// SetNats sets the nats connection for the Plugis instance.
func (plugis *Plugis) SetNats(nc *nats.Conn) {
	plugis.nc = nc
}

// Nats returns the nats connection for the Plugis instance.
func (plugis *Plugis) Nats() *nats.Conn {
	return plugis.nc
}

// Publish publishes a message to the nats connection.
func (plugis *Plugis) Publish(topic string, payload []byte) error {
	attrs := []slog.Attr{
		slog.String("fn", "Plugis.Publish"),
		slog.String("topic", topic),
		slog.String("payload", string(payload)),
	}
	if plugis.nc == nil {
		return ErrNatsConnectionNil
	}
	err := plugis.nc.Publish(topic, payload)
	if err != nil {
		attrs = append(attrs, slog.String("err", err.Error()))
		plugis.Logger().LogAttrs(context.TODO(), slog.LevelError, "Publish payload to topic", attrs...)
	} else {
		plugis.Logger().LogAttrs(context.TODO(), slog.LevelDebug, "Published payload to topic", attrs...)
	}
	return err
}

// Prefix returns the prefix for the Plugis instance.
func (plugis *Plugis) Prefix() string {
	return plugis.prefix
}

// SetPrefix sets the prefix for the Plugis instance.
func (plugis *Plugis) SetPrefix(prefix string) {
	plugis.prefix = prefix
}

// Request sends a request to the nats connection.
func (plugis *Plugis) Request(subj string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	attrs := []slog.Attr{
		slog.String("fn", "Plugis.Request"),
		slog.String("subject", subj),
		slog.String("data", string(data)),
		slog.Duration("timeout", timeout),
	}
	if plugis.nc == nil {
		attrs = append(attrs, slog.String("err", ErrNatsConnectionNil.Error()))
		plugis.Logger().LogAttrs(context.TODO(), slog.LevelError, "Request failed - nats connection is nil", attrs...)
		return nil, ErrNatsConnectionNil
	}
	msg, err := plugis.nc.Request(subj, data, timeout)
	if err != nil {
		attrs = append(attrs, slog.String("err", err.Error()))
		plugis.Logger().LogAttrs(context.TODO(), slog.LevelError, "Request failed", attrs...)
	} else {
		plugis.Logger().LogAttrs(context.TODO(), slog.LevelDebug, "Request successful", attrs...)
	}
	return msg, err
}

// RequestCtx sends a request to the nats connection and returns a single message.
// context is used for timeout and cancellation.
func (plugis *Plugis) RequestCtx(ctx context.Context, subj string, data []byte) (*nats.Msg, error) {
	return nil, errors.New("not implemented")
	deadline, ok := ctx.Deadline()
	if !ok {
		//ctx, _ = context.WithTimeout(ctx, time.Hour*24)
	} else {
		_ = deadline
	}
	iter, err := plugis.RequestMany(ctx, subj, data, natsext.RequestManyMaxMessages(1))
	plugis.Logger().Warn("RequestMany",
		slog.String("subject", subj),
		"err", err,
		"ctxErr", ctx.Err(),
		"cancealed", task.IsCancelled(ctx),
	)
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	for msg := range iter {
		return msg, nil
	}
	return &nats.Msg{}, ctx.Err()
}

// RequestMany sends a request to the nats connection and returns a sequence of messages.
func (plugis *Plugis) RequestMany(ctx context.Context, subject string, data []byte, opts ...natsext.RequestManyOpt) (iter.Seq2[*nats.Msg, error], error) {
	attrs := []slog.Attr{
		slog.String("fn", "Plugis.RequestMany"),
		slog.String("subject", subject),
		slog.String("data", string(data)),
	}
	if plugis.nc == nil {
		attrs = append(attrs, slog.String("err", ErrNatsConnectionNil.Error()))
		plugis.Logger().LogAttrs(context.TODO(), slog.LevelError, "RequestMany failed - nats connection is nil", attrs...)
		return nil, ErrNatsConnectionNil
	}
	result, err := natsext.RequestMany(ctx, plugis.nc, subject, data, opts...)
	if err != nil {
		attrs = append(attrs, slog.String("err", err.Error()))
		plugis.Logger().LogAttrs(context.TODO(), slog.LevelError, "RequestMany failed", attrs...)
	} else {
		plugis.Logger().LogAttrs(context.TODO(), slog.LevelDebug, "RequestMany successful", attrs...)
	}
	return result, err
}

// GetServices sends a request to the $SRV.INFO subject and returns a list of services.
func (plugis *Plugis) GetServices(ctx context.Context) ([]ServiceInfo, error) {
	iter, err := plugis.RequestMany(ctx, "$SRV.INFO", []byte(""))
	if err != nil {
		return nil, err
	}
	var services []ServiceInfo
	for msg := range iter {
		var serviceInfo ServiceInfo
		err := json.Unmarshal(msg.Data, &serviceInfo)
		if err != nil {
			return nil, err
		}
		services = append(services, serviceInfo)
	}
	return services, nil
}

// isRunningInDockerContainer
func isRunningInDockerContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

// NewServiceMetadata creates and fills a ServiceMetadata structure
func NewServiceMetadata(prefix string, startedAt time.Time) (*ServiceMetadata, error) {
	var err error
	var meta ServiceMetadata
	meta.Prefix = prefix
	meta.Platform = runtime.GOOS + "/" + runtime.GOARCH
	if isRunningInDockerContainer() {
		meta.Platform += " (docker)"
	}

	meta.StartedAt = startedAt.Format(time.RFC3339)
	meta.Hostname, err = os.Hostname()
	if err != nil {
		return nil, err
	}
	meta.MAC, err = net.GetMACAddress()
	if err != nil {
		return nil, err
	}

	return &meta, nil
}

//func (smd *ServiceMetadata) Meta() Metadata {
//	var meta Metadata
//	err := mapstructure.Decode(smd, &meta)
//	if err != nil {
//		return Metadata{}
//	}
//	return meta
//}

// Meta returns ServiceMetaData as map[string]string
func (smd *ServiceMetadata) Meta() Metadata {
	data, err := json.Marshal(smd)
	if err != nil {
		return Metadata{}
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}
	}
	return meta
}

// StartService initializes and starts a NATS service for the given PlugisServiceIntf implementation.
// It returns the created NatsService and any error encountered during the creation process.
func (plugis *Plugis) StartService(svc PlugisServiceIntf) (*nats_service.NatsService, error) {
	service, err := nats_service.NewNatsService(plugis.Nats(), plugis.Prefix(), micro.Config{
		Name:        svc.Name(),
		Endpoint:    nil,
		Version:     svc.Version(),
		Description: svc.Description(),
		Metadata:    svc.Metadata(),
	})
	return service, err
}

// VariableSet sets a variable with the given name, value, and type, then publishes it to a corresponding topic.
func (plugis *Plugis) VariableSet(name string, value any, varType string) error {
	variable := model.Variable{
		Name:    name,
		Value:   value,
		VarType: varType,
	}
	event := Event[model.Variable]{
		Type: "variable.set",
		Data: variable,
	}
	topic := "variable.set." + name
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return plugis.Publish(topic, payload)
}

// VariableUnset unsets a variable with the given name, then publishes it to a corresponding topic.
func (plugis *Plugis) VariableUnset(name string) error {
	variable := model.Variable{
		Name: name,
	}
	event := Event[model.Variable]{
		Type: "variable.unset",
		Data: variable,
	}
	topic := "variable.unset." + name
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return plugis.Publish(topic, payload)
}
