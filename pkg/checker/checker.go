package checker

import (
	"sync"
)

func nullHandler(buf []byte, userdata interface{}) bool {
	return true
}

type HealthResponse struct {
	URL          string
	AuthType     string
	Status       int
	ResponseTime float64
}

// Interface for publishers
type Interface interface {
	Check(urls []string, ch chan HealthResponse, wg *sync.WaitGroup)
}
