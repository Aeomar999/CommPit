package webhooks

import (
	"net/url"
	"os"
	"strings"
)

// dockerHost is the DNS name Docker provides for the host machine.
const dockerHost = "host.docker.internal"

// loopbackHosts are rewritten when running inside Docker: a callback URL
// pointing at them means the developer's app on the host, not this
// container.
var loopbackHosts = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
}

// InDocker reports whether mocksms runs inside a container (design spec
// §7.5: /.dockerenv or MOCKSMS_IN_DOCKER=1).
func InDocker() bool {
	if os.Getenv("MOCKSMS_IN_DOCKER") == "1" {
		return true
	}
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

// RewriteLoopback rewrites a loopback webhook host to Docker's host name.
// It returns the rewritten URL and true, or the input unchanged and false
// when there is nothing to rewrite (non-loopback host, unparseable URL).
// Port, path, query and fragment are preserved.
func RewriteLoopback(rawURL string) (string, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, false
	}
	host := parsed.Hostname()
	if host == "" || !loopbackHosts[strings.ToLower(host)] {
		return rawURL, false
	}
	if port := parsed.Port(); port != "" {
		parsed.Host = dockerHost + ":" + port
	} else {
		parsed.Host = dockerHost
	}
	return parsed.String(), true
}
