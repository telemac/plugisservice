package echoservice

import (
	"context"
	"github.com/telemac/plugisservice"
	"time"
)

// Ensure EchoService implements plugisservice.PlugisServiceIntf interface
var _ plugisservice.PlugisServiceIntf = (*EchoService)(nil)

// EchoService is a service that echoes a message.
type EchoService struct {
	plugisservice.Plugis
}

func NewEchoService() *EchoService {
	return &EchoService{}
}

// ExecuteCommand sends a command
func (svc *EchoService) ExecuteCommand(ctx context.Context, command string) ([]byte, error) {
	subject := "ism.homelab.service.plugis.command"

	svc.Logger().Info("sending command",
		"command", command,
		"subject", subject,
	)

	msg, err := svc.Request(subject, []byte(command), time.Second*5)
	if err != nil {
		svc.Logger().Error("command execution failed",
			"error", err,
			"command", command,
		)
		return nil, err
	} else {
		svc.Logger().Info("command executed successfully",
			"command", command,
			"response", string(msg.Data),
		)
	}

	return msg.Data, nil
}

// Run is the main function of the service.
func (svc *EchoService) Run(ctx context.Context) error {
	svc.Logger().Info("Run started")
	defer svc.Logger().Info("Run finished\a")

	//services, err := svc.GetServices(ctx)
	//svc.Logger().Info("services",
	//	"services", services,
	//	"error", err,
	//)

	svc.ExecuteCommand(ctx, "sleep 3")

	service, err := svc.StartService(svc)
	if err != nil {
		return err
	}
	defer func() {
		service.Stop()
	}()
	pingEndpoint.UserData = svc
	err = service.AddEndpoint(ctx, pingEndpoint)
	if err != nil {
		return err
	}

	<-ctx.Done()

	service.Stop()

	return nil
}

// Name returns the name of the service.
func (svc *EchoService) Name() string {
	return "echo"
}

// Description returns the description of the service.
func (svc *EchoService) Description() string {
	return "Echo service"
}

// Version returns the version of the service.
func (svc *EchoService) Version() string {
	return "1.0.0"
}

// Metadata returns the metadata of the service.
func (svc *EchoService) Metadata() plugisservice.Metadata {
	serviceMetadata, err := plugisservice.NewServiceMetadata(svc.Prefix(), time.Now())
	if err != nil {
		svc.Logger().Error("NewServiceMetadata", "error", err)
	}
	return serviceMetadata.Meta()
}
