package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	flags "github.com/jessevdk/go-flags"
	"github.com/verdel/squid-auth-healthcheck/pkg/checker"
	"github.com/verdel/squid-auth-healthcheck/pkg/slice"
)

const (
	version = "0.0.4"
)

var opts struct {
	URL               []string `short:"u" long:"url" description:"url to check for availability (required)" required:"true"`
	AuthType          []string `long:"auth-type" description:"type of used proxy authentication mechanism. [ntlm, kerberos, no, all] (required)" required:"true"`
	ProxyAddr         string   `long:"proxy-addr" description:"proxy server address (required)" required:"true"`
	ProxyPort         int      `long:"proxy-port" description:"proxy server port (default: 3128)" default:"3128"`
	ProxyUsername     string   `long:"proxy-username" description:"proxy user login"`
	ProxyPassword     string   `long:"proxy-password" description:"proxy user password"`
	ConnectionTimeout int      `long:"timeout" description:"healthcheck connection timeout in seconds (default: 2)" default:"2"`
	StrictURL         bool     `long:"strict-url" description:"the check returns a positive result only if all URLs are available"`
	StrictAuth        bool     `long:"strict-auth" description:"the check returns a positive result only if url are available with all auth method"`
	ClusterCheck      bool     `long:"cluster-check" description:"check through proxy cluster node instead standalone proxy server"`
	IngressProxyAddr  string   `long:"ingress-proxy-addr" description:"ingress proxy address. It will be used for kerberos verification. This FQDN will be used when forming the request, but the IP address of the node of the proxy server cluster will be used as the IP address" required:"false"`
	Verbose           bool     `short:"v" long:"verbose" description:"output verbose healthcheck information"`
}
var allowAuthType = []string{"ntlm", "kerberos", "no", "all"}

func exitOK(verbose bool) {
	if !verbose {
		fmt.Println(1)
	}
	os.Exit(0)
}

func exitErr(verbose bool) {
	if !verbose {
		fmt.Println(0)
	}
	os.Exit(1)
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	parser.Usage = fmt.Sprintf("\n\nVersion: %s", version)

	if len(os.Args) == 1 {
		parser.WriteHelp(os.Stderr)
		os.Exit(0)
	}
	_, err := parser.Parse()

	if err != nil {
		os.Exit(1)
	}

	if len(opts.AuthType) > 1 || (len(opts.AuthType) == 1 && !slice.StringInSlice("no", opts.AuthType)) {
		if opts.ProxyPassword == "" || opts.ProxyUsername == "" {
			fmt.Println("the required flags `--proxy-username' and `--proxy-password' were not specified")
			os.Exit(1)
		}
	}

	if opts.ClusterCheck && opts.IngressProxyAddr == "" {
		fmt.Println("the required flags `--ingress-proxy-addr' were not specified")
		os.Exit(1)
	}

	var authType []string
	if slice.StringInSlice("all", opts.AuthType) {
		for _, v := range allowAuthType {
			if v != "all" {
				authType = append(authType, v)
			}
		}
	} else {
		authType = make([]string, len(opts.AuthType))
		copy(authType, opts.AuthType)
	}

	if len(authType) > len(allowAuthType) {
		fmt.Println("Too many authentication type")
		os.Exit(1)
	}

	for _, item := range authType {
		if !slice.StringInSlice(item, allowAuthType) {
			fmt.Printf("Authentication type %s is not allowed", item)
			os.Exit(1)
		}
	}

	var wg sync.WaitGroup

	ch := make(chan checker.HealthResponse, len(authType)*len(opts.URL))
	wg.Add(len(authType))

	if slice.StringInSlice("ntlm", authType) {
		var ntlm checker.Interface = checker.NewAuthNTLM(opts.ProxyAddr, opts.ProxyPort, opts.ProxyUsername, opts.ProxyPassword, opts.ConnectionTimeout)
		go ntlm.Check(opts.URL, ch, &wg)
	}
	if slice.StringInSlice("kerberos", authType) {
		var IngressProxyAddr string

		if opts.ClusterCheck {
			IngressProxyAddr = opts.IngressProxyAddr
		} else {
			IngressProxyAddr = opts.ProxyAddr
		}

		kerberos, err := checker.NewAuthKerberos(IngressProxyAddr, opts.ProxyAddr, opts.ProxyPort, opts.ProxyUsername, opts.ProxyPassword, opts.ConnectionTimeout)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		go kerberos.Check(opts.URL, ch, &wg)
	}
	if slice.StringInSlice("no", authType) {
		var no checker.Interface = checker.NewAuthNo(opts.ProxyAddr, opts.ProxyPort, opts.ConnectionTimeout)
		go no.Check(opts.URL, ch, &wg)
	}

	wg.Wait()
	close(ch)

	var result []checker.HealthResponse

	var okURLResult []string
	var okAuthResult []string

	for response := range ch {
		if response.Status == 1 {
			okURLResult = slice.AppendIfMissing(okURLResult, response.URL)
			okAuthResult = slice.AppendIfMissing(okAuthResult, response.AuthType)
		}
		result = append(result, response)
	}

	if opts.Verbose {
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(result)
	}

	if opts.StrictAuth {
		if len(okAuthResult) < len(authType) {
			exitErr(opts.Verbose)
		}
	}

	if opts.StrictURL {
		if len(okURLResult) < len(opts.URL) {
			exitErr(opts.Verbose)
		}
	} else {
		if len(okURLResult) == 0 {
			exitErr(opts.Verbose)
		}
	}
	exitOK(opts.Verbose)
}
