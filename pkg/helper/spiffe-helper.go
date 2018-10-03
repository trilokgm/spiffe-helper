package helper

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/apex/log"
	"github.com/spiffe/spire/api/workload"
	proto "github.com/spiffe/spire/proto/api/workload"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"spiffe-helper/cmd/config"
	"strings"
	"sync/atomic"
	"syscall"
)

// sidecar is the component that consumes the Workload API and renews certs
// implements the interface Sidecar
type Sidecar struct {
	Config            *config.SidecarConfig
	ProcessRunning    int32
	Process           *os.Process
	WorkloadAPIClient workload.X509Client
}

const (
	certsFileMode = os.FileMode(0644)
	keyFileMode   = os.FileMode(0600)
)

// RunDaemon starts the main loop
// Starts the workload API client to listen for new SVID updates
// When a new SVID is received on the updateChan, the SVID certificates
// are stored in disk and a restart signal is sent to the proxy's process
func (s *Sidecar) RunDaemon(ctx context.Context) error {
	// Create channel for interrupt signal
	interrupt := make(chan os.Signal, 1)
	errorChan := make(chan error, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	updateChan := s.WorkloadAPIClient.UpdateChan()

	//start the WorkloadAPIClient
	go func() {
		err := s.WorkloadAPIClient.Start()
		if err != nil {
			log.Error(err.Error())
			errorChan <- err
		}
	}()
	defer s.WorkloadAPIClient.Stop()

	for {
		select {
		case svidResponse := <-updateChan:
			updateCertificates(s, svidResponse)
		case <-interrupt:
			return nil
		case err := <-errorChan:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

// Updates the certificates stored in disk and signal the Process to restart
func updateCertificates(s *Sidecar, svidResponse *proto.X509SVIDResponse) {
	err := s.dumpBundles(svidResponse)
	if err != nil {
		log.Error(err.Error())
		return
	}
	err = s.signalProcess()
	if err != nil {
		log.Error(err.Error())
	}
}

//signalProcess sends the configured Renew signal to the process running the proxy
//to reload itself so that the proxy uses the new SVID
func (s *Sidecar) signalProcess() (err error) {
	if atomic.LoadInt32(&s.ProcessRunning) == 0 {
		cmd := exec.Command(s.Config.Cmd, strings.Split(s.Config.CmdArgs, " ")...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			return fmt.Errorf("error executing process: %v\n%v", s.Config.Cmd, err)
		}
		s.Process = cmd.Process
		go s.checkProcessExit()
	} else {
		// Signal to reload certs
		sig, err := getSignal(s.Config.RenewSignal)
		if err != nil {
			return fmt.Errorf("error getting signal: %v\n%v", s.Config.RenewSignal, err)
		}

		err = s.Process.Signal(sig)
		if err != nil {
			return fmt.Errorf("error signaling process with signal: %v\n%v", sig, err)
		}
	}

	return nil
}

func (s *Sidecar) checkProcessExit() {
	atomic.StoreInt32(&s.ProcessRunning, 1)
	s.Process.Wait()
	atomic.StoreInt32(&s.ProcessRunning, 0)
}

//dumpBundles takes a X509SVIDResponse, representing a svid message from
//the Workload API, and calls writeCerts and writeKey to write to disk
//the svid, key and bundle of certificates
func (s *Sidecar) dumpBundles(svidResponse *proto.X509SVIDResponse) error {

	// There may be more than one certificate, but we are interested in the first one only
	svid := svidResponse.Svids[0]

	svidFile := path.Join(s.Config.CertDir, s.Config.SvidFileName)
	svidKeyFile := path.Join(s.Config.CertDir, s.Config.SvidKeyFileName)
	svidBundleFile := path.Join(s.Config.CertDir, s.Config.SvidBundleFileName)

	err := s.writeCerts(svidFile, svid.X509Svid)
	if err != nil {
		return err
	}

	err = s.writeKey(svidKeyFile, svid.X509SvidKey)
	if err != nil {
		return err
	}

	err = s.writeCerts(svidBundleFile, svid.Bundle)
	if err != nil {
		return err
	}

	return nil
}

// writeCerts takes a slice of bytes, which may contain multiple certificates,
// and encodes them as PEM blocks, writing them to file
func (s *Sidecar) writeCerts(file string, data []byte) error {
	certs, err := x509.ParseCertificates(data)
	if err != nil {
		return err
	}

	pemData := []byte{}
	for _, cert := range certs {
		b := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}

	return ioutil.WriteFile(file, pemData, certsFileMode)
}

// writeKey takes a private key as a slice of bytes,
// formats as PEM, and writes it to file
func (s *Sidecar) writeKey(file string, data []byte) error {
	b := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: data,
	}

	return ioutil.WriteFile(file, pem.EncodeToMemory(b), keyFileMode)
}

func getSignal(s string) (sig syscall.Signal, err error) {
	switch s {
	case "SIGABRT":
		sig = syscall.SIGABRT
	case "SIGALRM":
		sig = syscall.SIGALRM
	case "SIGBUS":
		sig = syscall.SIGBUS
	case "SIGCHLD":
		sig = syscall.SIGCHLD
	case "SIGCONT":
		sig = syscall.SIGCONT
	case "SIGFPE":
		sig = syscall.SIGFPE
	case "SIGHUP":
		sig = syscall.SIGHUP
	case "SIGILL":
		sig = syscall.SIGILL
	case "SIGIO":
		sig = syscall.SIGIO
	case "SIGIOT":
		sig = syscall.SIGIOT
	case "SIGKILL":
		sig = syscall.SIGKILL
	case "SIGPIPE":
		sig = syscall.SIGPIPE
	case "SIGPROF":
		sig = syscall.SIGPROF
	case "SIGQUIT":
		sig = syscall.SIGQUIT
	case "SIGSEGV":
		sig = syscall.SIGSEGV
	case "SIGSTOP":
		sig = syscall.SIGSTOP
	case "SIGSYS":
		sig = syscall.SIGSYS
	case "SIGTERM":
		sig = syscall.SIGTERM
	case "SIGTRAP":
		sig = syscall.SIGTRAP
	case "SIGTSTP":
		sig = syscall.SIGTSTP
	case "SIGTTIN":
		sig = syscall.SIGTTIN
	case "SIGTTOU":
		sig = syscall.SIGTTOU
	case "SIGURG":
		sig = syscall.SIGURG
	case "SIGUSR1":
		sig = syscall.SIGUSR1
	case "SIGUSR2":
		sig = syscall.SIGUSR2
	case "SIGVTALRM":
		sig = syscall.SIGVTALRM
	case "SIGWINCH":
		sig = syscall.SIGWINCH
	case "SIGXCPU":
		sig = syscall.SIGXCPU
	case "SIGXFSZ":
		sig = syscall.SIGXFSZ
	default:
		err = fmt.Errorf("unrecognized signal: %v", s)
	}

	return sig, err
}
