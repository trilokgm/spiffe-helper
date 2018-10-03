package helper

import (
	"context"
	"crypto/x509"
	"github.com/spiffe/spire/proto/api/workload"
	"github.com/spiffe/spire/test/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path"
	"spiffe-helper/cmd/config"
	"sync"
	"testing"
	"time"
)

//Creates a Sidecar with a Mocked WorkloadAPIClient and tests that
//running the Sidecar Daemon, when a SVID Response is sent to the
//UpdateChan on the WorkloadAPI client, the PEM files are stored on disk
func TestSidecar_RunDaemon(t *testing.T) {

	var wg sync.WaitGroup

	tmpdir, err := ioutil.TempDir("", "sidecar-run-daemon")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	config := &config.SidecarConfig{
		Cmd:                "echo",
		CertDir:            tmpdir,
		SvidFileName:       "svid.pem",
		SvidKeyFileName:    "svid_key.pem",
		SvidBundleFileName: "svid_bundle.pem",
	}

	updateMockChan := make(chan *workload.X509SVIDResponse)
	workloadClient := MockWorkloadClient{
		mockChan: updateMockChan,
	}

	sidecar := sidecar{
		config:            config,
		WorkloadAPIClient: workloadClient,
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = sidecar.RunDaemon(ctx)
		require.NoError(t, err)
	}()

	x509SvidTestResponse := x509SvidResponse(t)

	//send a X509SVIDResponse to Updates channel
	updateMockChan <- x509SvidTestResponse

	//send signal to stop the Daemon
	cancel()
	wg.Wait()

	//The expected files
	svidFile := path.Join(tmpdir, config.SvidFileName)
	svidKeyFile := path.Join(tmpdir, config.SvidKeyFileName)
	svidBundleFile := path.Join(tmpdir, config.SvidBundleFileName)

	if _, err := os.Stat(svidFile); err != nil {
		t.Errorf("error %v with file: %v", err, svidFile)
	}
	if _, err := os.Stat(svidKeyFile); err != nil {
		t.Errorf("error %v with file: %v", err, svidKeyFile)
	}
	if _, err := os.Stat(svidBundleFile); err != nil {
		t.Errorf("error %v with file: %v", err, svidBundleFile)
	}
}

//Tests that when there is no defaultTimeout in the config, it uses
//the default defaultTimeout set in a constant in the spiffe_helper
func Test_getTimeout_default(t *testing.T) {
	config := &config.SidecarConfig{}

	expectedTimeout := defaultTimeout
	actualTimeout, err := getTimeout(config)

	assert.NoError(t, err)
	if actualTimeout != expectedTimeout {
		t.Errorf("Expected defaultTimeout : %v, got %v", expectedTimeout, actualTimeout)
	}
}

//Tests that when there is a timeout set in the config, it's used that one
func Test_getTimeout_custom(t *testing.T) {
	config := &config.SidecarConfig{
		Timeout: "10s",
	}

	expectedTimeout := time.Second * 10
	actualTimeout, err := getTimeout(config)

	assert.NoError(t, err)
	if actualTimeout != expectedTimeout {
		t.Errorf("Expected defaultTimeout : %v, got %v", expectedTimeout, actualTimeout)
	}
}

func Test_getTimeout_return_error_when_parsing_fails(t *testing.T) {
	config := &config.SidecarConfig{
		Timeout: "invalid",
	}

	actualTimeout, err := getTimeout(config)

	assert.Empty(t, actualTimeout)
	assert.NotEmpty(t, err)
}

type MockWorkloadClient struct {
	mockChan chan *workload.X509SVIDResponse
}

func (m MockWorkloadClient) Start() error {
	return nil
}

func (m MockWorkloadClient) Stop() {}

func (m MockWorkloadClient) UpdateChan() <-chan *workload.X509SVIDResponse {
	return m.mockChan
}

// creates a X509SVIDResponse reading test certs from files
func x509SvidResponse(t *testing.T) *workload.X509SVIDResponse {
	svid, key, err := util.LoadSVIDFixture()
	if err != nil {
		t.Errorf("could not load svid fixture: %v", err)
	}
	ca, _, err := util.LoadCAFixture()
	if err != nil {
		t.Errorf("could not load ca fixture: %v", err)
	}

	keyData, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Errorf("could not marshal private key: %v", err)
	}

	svidMsg := &workload.X509SVID{
		SpiffeId:    "spiffe://example.org/foo",
		X509Svid:    svid.Raw,
		X509SvidKey: keyData,
		Bundle:      ca.Raw,
	}
	return &workload.X509SVIDResponse{
		Svids: []*workload.X509SVID{svidMsg},
	}
}
