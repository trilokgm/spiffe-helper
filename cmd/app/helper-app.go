package app

import (
	"context"
	"github.com/spiffe/spire/api/workload"
	"net"
	"spiffe-helper/cmd/config"
	"spiffe-helper/pkg/helper"
	"time"
)

const (
	// default timeout Duration for the workloadAPI client when the defaultTimeout
	// is not configured in the .conf file
	defaultTimeout = time.Duration(5 * time.Second)
)

type HelperApp struct {

}
// NewSidecar creates a new sidecar
func (h *HelperApp) NewSidecar(config *config.SidecarConfig) (*helper.Sidecar, error) {
	timeout, err := getTimeout(config)
	if err != nil {
		return nil, err
	}

	return &helper.Sidecar{
		Config:            config,
		WorkloadAPIClient: newWorkloadAPIClient(config.AgentAddress, timeout),
	}, nil
}

func (h *HelperApp) NewContext() context.Context {
	return context.Background()
}

func (h *HelperApp) Start(ctx context.Context, helper *helper.Sidecar) error {
	return helper.RunDaemon(ctx)
}

// parses a time.Duration from the the SidecarConfig,
// if there's an error during parsing, maybe because
// it's not well defined or not defined at all in the
// config, returns the defaultTimeout constant
func getTimeout(config *config.SidecarConfig) (time.Duration, error) {
	if config.Timeout == "" {
		return defaultTimeout, nil
	}

	t, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return 0, err
	}
	return t, nil
}

//newWorkloadAPIClient creates a workload.X509Client
func newWorkloadAPIClient(agentAddress string, timeout time.Duration) workload.X509Client {
	addr := &net.UnixAddr{
		Net:  "unix",
		Name: agentAddress,
	}
	config := &workload.X509ClientConfig{
		Addr:    addr,
		Timeout: timeout,
	}
	return workload.NewX509Client(config)
}
