package checker

import (
	"sync"

	curl "github.com/andelf/go-curl"
)

type AuthNo struct {
	authType          string
	ProxyAddr         string
	ProxyPort         int
	ConnectionTimeout int
}

func NewAuthNo(ProxyAddr string, ProxyPort int, ConnectionTimeout int) *AuthNo {
	var a AuthNo
	a.authType = "no"
	a.ProxyAddr = ProxyAddr
	a.ProxyPort = ProxyPort
	a.ConnectionTimeout = ConnectionTimeout
	return &a
}

func (a *AuthNo) Check(urls []string, ch chan HealthResponse, wg *sync.WaitGroup) {
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
