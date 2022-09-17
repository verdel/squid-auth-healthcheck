# squid-auth-healthcheck

This application checks the availability of the URL using a proxy server with different types of authorization (NTLM, Kerberos).
The application returns 0 or 1, depending on the input conditions and the availability of the URL.
If you use the `--verbose` flag, it returns json with the availability status and also the full response time for each URL

## Building

The service based on go-curl and required libcurl development files. Use your OS package manager to install libcurl-devel or something like this.

## Running

The user name must be entered in the UPN(User principal name) format. The domain name must be typed in capital letters.

### Example

```
squid-auth-healthcheck --proxy-addr 127.0.0.1 --proxy-username test@TEST.LOCAL --proxy-password test --url https://google.com --strict-url --strict-auth --auth-type ntlm --auth-type kerberos --verbose
```

You can also check the availability of a resource on the Internet by accessing a separate node of the proxy server cluster using the address of the proxy server cluster

```
squid-auth-healthcheck --cluster-check --ingress-proxy-addr proxy.contoso.com --proxy-addr proxy-node01.contoso.com --proxy-username test@TEST.LOCAL --proxy-password test --url https://google.com --strict-url --strict-auth --auth-type ntlm --auth-type kerberos --verbose
```

## Parameters

```
Application Options:
  -u, --url=                url to check for availability (required)
      --auth-type=          type of used proxy authentication mechanism. [ntlm, kerberos, no, all] (required)
      --proxy-addr=         proxy server address (required)
      --proxy-port=         proxy server port (default: 3128)
      --proxy-username=     proxy user login
      --proxy-password=     proxy user password
      --timeout=            healthcheck connection timeout in seconds (default: 2)
      --strict-url          the check returns a positive result only if all URLs are available
      --strict-auth         the check returns a positive result only if url are available with all auth method
      --cluster-check       check through proxy cluster node instead standalone proxy server
      --ingress-proxy-addr= ingress proxy address. It will be used for kerberos verification. This FQDN will be used when forming the request, but the IP address of the node of the proxy server cluster will be used as the IP address
  -v, --verbose             output verbose healthcheck information
```
