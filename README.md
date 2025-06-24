# plugisservice

plugisservice is a package to write plugis3 services in Go.

A Plugis3 service is a NATS micro service with some specific properties that must implement the `PlugisServiceIntf` interface.

## NATS Micro PlugisServiceIntf Integration

For NATS services, see:
- [Building a Service](https://docs.nats.io/using-nats/nex/getting-started/building-service)
- [NATS micro on Github](https://github.com/nats-io/nats.go/tree/main/micro)

# Files

The `plugisservice.go` file defines the `PlugisServiceIntf` interface, which specifies the contract that all plugis services must implement.

The `plugis.go` file defines the `PlugisIntf` interface, which specifies the functions provided by plugis

The `example/echoService/echoService.go` file is a minimal service implementation sample.

The `example/echoService/cmd/main.go` file is a sample of ServiceRunner usage.