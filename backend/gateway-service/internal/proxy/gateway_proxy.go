package proxy

import (
	"bytes"
	"io"
	"net/http"
)

// GatewayProxy handles proxying requests to backend services
type GatewayProxy struct {
	authServiceURL     string
	workflowServiceURL string
	workerServiceURL   string
	eventServiceURL    string
	client             *http.Client
}

// NewGatewayProxy creates a new gateway proxy
func NewGatewayProxy(authURL, workflowURL, workerURL, eventURL string) *GatewayProxy {
	return &GatewayProxy{
		authServiceURL:     authURL,
		workflowServiceURL: workflowURL,
		workerServiceURL:   workerURL,
		eventServiceURL:    eventURL,
		client: &http.Client{
			Timeout: 10 * 1000 * 1000 * 1000, // 10 seconds
		},
	}
}

// ProxyRequest proxies a request to a backend service
func (gp *GatewayProxy) ProxyRequest(method, path, serviceURL string, headers http.Header, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, serviceURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Copy headers from original request
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	return gp.client.Do(req)
}

// ProxyToAuthService proxies a request to the auth service
func (gp *GatewayProxy) ProxyToAuthService(method, path string, headers http.Header, body []byte) (*http.Response, error) {
	return gp.ProxyRequest(method, path, gp.authServiceURL, headers, body)
}

// ProxyToWorkflowService proxies a request to the workflow service
func (gp *GatewayProxy) ProxyToWorkflowService(method, path string, headers http.Header, body []byte) (*http.Response, error) {
	return gp.ProxyRequest(method, path, gp.workflowServiceURL, headers, body)
}

// ProxyToWorkerService proxies a request to the worker service
func (gp *GatewayProxy) ProxyToWorkerService(method, path string, headers http.Header, body []byte) (*http.Response, error) {
	return gp.ProxyRequest(method, path, gp.workerServiceURL, headers, body)
}

// ProxyToEventService proxies a request to the event service
func (gp *GatewayProxy) ProxyToEventService(method, path string, headers http.Header, body []byte) (*http.Response, error) {
	return gp.ProxyRequest(method, path, gp.eventServiceURL, headers, body)
}
