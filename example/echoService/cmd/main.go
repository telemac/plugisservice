package main

import (
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/telemac/plugisservice"

	echoservice "github.com/telemac/plugisservice/example/echoService"

	"github.com/telemac/goutils/task"
)

func main() {
	ctx, cancel := task.NewCancellableContext(time.Second * 10)
	defer cancel()
	logger := slog.Default().With("service", "echoService")

	nc, err := nats.Connect("wss://idronebox:admin@n1.idronebox.com")
	if err != nil {
		logger.Error("connect to nat", "err", err)
		return
	}
	defer nc.Close()

	runner := plugisservice.NewServiceRunner(nc, logger, "integrator.customer.instance")

	runner.Start(ctx, echoservice.NewEchoService())

	runner.Wait()

}
