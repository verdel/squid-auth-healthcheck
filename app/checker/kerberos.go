package checker

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"

	curl "github.com/andelf/go-curl"
)

const (
	kinitCmd    = "kinit"
	kdestroyCmd = "kdestroy"
)

type AuthKerberos struct {
	authType          string
	ProxyAddr         string
	ProxyPort         int
	ProxyUsername     string
	ProxyPassword     string
	ConnectionTimeout int
}

func NewAuthKerberos(ProxyAddr string, ProxyPort int, ProxyUsername string, ProxyPassword string, ConnectionTimeout int) (*AuthKerberos, error) {
	var a AuthKerberos
	a.authType = "kerberos"
	a.ProxyAddr = ProxyAddr
	a.ProxyPort = ProxyPort
	a.ProxyUsername = ProxyUsername
	a.ProxyPassword = ProxyPassword
	a.ConnectionTimeout = ConnectionTimeout

	if err := a.loginKRB(ProxyUsername, ProxyPassword); err != nil {
		return nil, err
	} else {
		return &a, nil
	}

}

func (a *AuthKerberos) loginKRB(username, password string) error {
	cmd := exec.Command(kinitCmd, username)

	stdinR, stdinW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	cmd.Stdin = stdinR
	cmd.Stderr = stderrW

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("could not start %s command: %v", kinitCmd, err)
	}

	go func() {
		io.WriteString(stdinW, password)
		stdinW.Close()
	}()
	errBuf := new(bytes.Buffer)
	go func() {
		io.Copy(errBuf, stderrR)
		stderrR.Close()
	}()

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("%s did not run successfully: %v stderr: %s", kinitCmd, err, string(errBuf.Bytes()))
	}
	return nil
}

func (a *AuthKerberos) destroyKRB() error {
	cmd := exec.Command(kdestroyCmd)
	stderrR, stderrW := io.Pipe()
	cmd.Stderr = stderrW
	cmd.Start()
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("could not start %s command: %v", kinitCmd, err)
	}

	errBuf := new(bytes.Buffer)
	go func() {
		io.Copy(errBuf, stderrR)
		stderrR.Close()
	}()

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("%s did not run successfully: %v stderr: %s", kinitCmd, err, string(errBuf.Bytes()))
	}
	return nil
}

func (a *AuthKerberos) Check(urls []string, ch chan HealthResponse, wg *sync.WaitGroup) {
	defer a.destroyKRB()
	var innerWg sync.WaitGroup
	innerWg.Add(len(urls))
	for _, url := range urls {
		go func(u string) {
			conn := curl.EasyInit()
			conn.Setopt(curl.OPT_VERBOSE, 0)
			conn.Setopt(curl.OPT_FOLLOWLOCATION, 1)
			conn.Setopt(curl.OPT_PROXYTYPE, curl.PROXY_HTTP)
			conn.Setopt(curl.OPT_PROXY, a.ProxyAddr)
			conn.Setopt(curl.OPT_PROXYPORT, a.ProxyPort)
			conn.Setopt(curl.OPT_PROXYAUTH, curl.AUTH_GSSNEGOTIATE)
			conn.Setopt(curl.OPT_PROXYUSERNAME, "")
			conn.Setopt(curl.OPT_PROXYPASSWORD, "")
			conn.Setopt(curl.OPT_TIMEOUT, a.ConnectionTimeout)
			conn.Setopt(curl.OPT_WRITEFUNCTION, nullHandler)
			conn.Setopt(curl.OPT_URL, u)
			if err := conn.Perform(); err != nil {
				ch <- HealthResponse{u, a.authType, 0, 0}
			} else {
				code, _ := conn.Getinfo(curl.INFO_RESPONSE_CODE)
				responseTime, _ := conn.Getinfo(curl.INFO_TOTAL_TIME)
				if code.(int) == 200 {
					ch <- HealthResponse{u, a.authType, 1, responseTime.(float64)}
				} else {
					ch <- HealthResponse{u, a.authType, 0, responseTime.(float64)}
				}
			}
			conn.Cleanup()
			innerWg.Done()
		}(url)
	}
	innerWg.Wait()
	wg.Done()
}
