package artifacts

import (
	"crypto/tls"
	"net/http"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

var (
	insecureHTTPClient *http.Client
)

func init() {
	insecureTransport := http.DefaultTransport.(*http.Transport).Clone()
	insecureTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	insecureHTTPClient = &http.Client{Transport: retry.NewTransport(insecureTransport)}
}

func newInsecureAuthClient() *auth.Client {
	return &auth.Client{
		Client: insecureHTTPClient,
		Header: http.Header{
			"User-Agent": {"oras-go"},
		},
		Cache: auth.DefaultCache,
	}
}
