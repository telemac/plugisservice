package echoservice

import (
	"context"
	"github.com/nats-io/nats.go/micro"
	nats_service "github.com/telemac/plugisservice/pkg/nats-service"
	"time"
)

var pingEndpoint = nats_service.EndpointConfig{
	Name: "ping",
	Handler: func(ctx context.Context, request micro.Request, ec nats_service.EndpointConfig) (any, error) {
		data := request.Data()
		_ = data
		// get plugis from EndpointConfigt
		echoService, ok := ec.UserData.(*EchoService)
		if ok {
			echoService.Logger().Info("plugis ping received")
			res, err := echoService.ExecuteCommand(ctx, "hostnamectl")
			return string(res), err
		}
		return ec, nil
		return "ping: " + string(data), nil
	},
	MaxConcurrency: 10,
	RequestTimeout: 2 * time.Second,
	Metadata: map[string]string{
		"description": "ping",
		"version":     "0.0.1",
	},
}
