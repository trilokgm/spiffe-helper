package main

import (
	"fmt"
	"spiffe-helper/cmd/app"
	"spiffe-helper/cmd/config"
)
func main() {

	fmt.Printf(" Test app 1 - register workload api client \n")
	config := config.SidecarConfig{
		AgentAddress: "/tmp/agent.sock",
		CertDir: "/opt/test-app1/certs/",
		RenewSignal: "SIGUSR1",
		SvidFileName: "svid.pem",
		SvidBundleFileName: "svid_bundle.pem",
		SvidKeyFileName: "svid.key",
	}
	sidecar, err := app.NewSidecar(&config)
	if err != nil {
		fmt.Printf("failed to create new sidecar [%s] \n,Exiting ...",err.Error())
		return
	}
	ctx := app.NewContext()
	err = app.Start(ctx, sidecar)
	if err != nil {
		fmt.Printf("failed to start helper [%s] \n",err.Error())
	} else {
		dummyChannel := make(chan struct{})
		fmt.Printf("waiting indefinitely on empty channel \n")
		<-dummyChannel
	}
	fmt.Printf("Exiting ..")
}
