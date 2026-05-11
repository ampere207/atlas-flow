package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"atlasflow/backend/gateway-service/internal/proxy"
	"atlasflow/backend/shared/middleware"
)

// GatewayHandler handles gateway routes
type GatewayHandler struct {
	proxy *proxy.GatewayProxy
}

// NewGatewayHandler creates a new gateway handler
func NewGatewayHandler(proxy *proxy.GatewayProxy) *GatewayHandler {
	return &GatewayHandler{proxy: proxy}
}

// SignUp proxies signup request to auth service
func (gh *GatewayHandler) SignUp(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	resp, err := gh.proxy.ProxyToAuthService(http.MethodPost, "/auth/signup", c.Request.Header, body)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach auth service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// Login proxies login request to auth service
func (gh *GatewayHandler) Login(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	resp, err := gh.proxy.ProxyToAuthService(http.MethodPost, "/auth/login", c.Request.Header, body)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach auth service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// RefreshToken proxies refresh token request to auth service
func (gh *GatewayHandler) RefreshToken(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	resp, err := gh.proxy.ProxyToAuthService(http.MethodPost, "/auth/refresh", c.Request.Header, body)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach auth service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// CreateWorkflow proxies workflow creation to workflow service
func (gh *GatewayHandler) CreateWorkflow(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodPost, "/workflows", c.Request.Header, body)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// GetWorkflow proxies workflow retrieval to workflow service
func (gh *GatewayHandler) GetWorkflow(c *gin.Context) {
	id := c.Param("id")
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodGet, "/workflows/"+id, c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// ListWorkflows proxies workflow list to workflow service
func (gh *GatewayHandler) ListWorkflows(c *gin.Context) {
	limit := c.Query("limit")
	offset := c.Query("offset")
	path := "/workflows"
	if limit != "" || offset != "" {
		path += "?"
		if limit != "" {
			path += "limit=" + limit
		}
		if offset != "" {
			if limit != "" {
				path += "&"
			}
			path += "offset=" + offset
		}
	}
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodGet, path, c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// UpdateWorkflowStatus proxies workflow status update to workflow service
func (gh *GatewayHandler) UpdateWorkflowStatus(c *gin.Context) {
	id := c.Param("id")
	body, _ := io.ReadAll(c.Request.Body)
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodPut, "/workflows/"+id+"/status", c.Request.Header, body)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// ExecuteWorkflow proxies workflow execution to workflow service.
func (gh *GatewayHandler) ExecuteWorkflow(c *gin.Context) {
	id := c.Param("id")
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodPost, "/workflows/"+id+"/execute", c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// CancelWorkflow proxies cancellation to workflow service.
func (gh *GatewayHandler) CancelWorkflow(c *gin.Context) {
	id := c.Param("id")
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodPost, "/workflows/"+id+"/cancel", c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// ListWorkflowTasks proxies task listing to workflow service.
func (gh *GatewayHandler) ListWorkflowTasks(c *gin.Context) {
	id := c.Param("id")
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodGet, "/workflows/"+id+"/tasks", c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// ListWorkflowHistory proxies history retrieval to workflow service.
func (gh *GatewayHandler) ListWorkflowHistory(c *gin.Context) {
	id := c.Param("id")
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodGet, "/workflows/"+id+"/history", c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// GetWorkflowExecutionStatus proxies status retrieval to workflow service.
func (gh *GatewayHandler) GetWorkflowExecutionStatus(c *gin.Context) {
	id := c.Param("id")
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodGet, "/workflows/"+id+"/status", c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// StreamWorkflowExecution proxies the workflow SSE stream.
func (gh *GatewayHandler) StreamWorkflowExecution(c *gin.Context) {
	id := c.Param("id")
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodGet, "/workflows/"+id+"/stream", c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// RegisterWorker proxies worker registration to worker service
func (gh *GatewayHandler) RegisterWorker(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	resp, err := gh.proxy.ProxyToWorkerService(http.MethodPost, "/workers", c.Request.Header, body)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach worker service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// GetWorker proxies worker retrieval to worker service
func (gh *GatewayHandler) GetWorker(c *gin.Context) {
	id := c.Param("id")
	resp, err := gh.proxy.ProxyToWorkerService(http.MethodGet, "/workers/"+id, c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach worker service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// ListWorkers proxies worker list to workflow service (where orchestrator manages workers)
func (gh *GatewayHandler) ListWorkers(c *gin.Context) {
	limit := c.Query("limit")
	offset := c.Query("offset")
	path := "/workers"
	if limit != "" || offset != "" {
		path += "?"
		if limit != "" {
			path += "limit=" + limit
		}
		if offset != "" {
			if limit != "" {
				path += "&"
			}
			path += "offset=" + offset
		}
	}
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodGet, path, c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// RecordHeartbeat proxies heartbeat to worker service
func (gh *GatewayHandler) RecordHeartbeat(c *gin.Context) {
	id := c.Param("id")
	body, _ := io.ReadAll(c.Request.Body)
	resp, err := gh.proxy.ProxyToWorkerService(http.MethodPost, "/workers/"+id+"/heartbeat", c.Request.Header, body)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach worker service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// GetClusterMetrics proxies cluster metrics to workflow service
func (gh *GatewayHandler) GetClusterMetrics(c *gin.Context) {
	resp, err := gh.proxy.ProxyToWorkflowService(http.MethodGet, "/cluster/metrics", c.Request.Header, nil)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadGateway, "failed to reach workflow service")
		return
	}
	gh.copyResponse(c.Writer, resp)
}

// copyResponse copies the backend response to the gateway response
func (gh *GatewayHandler) copyResponse(w http.ResponseWriter, resp *http.Response) {
	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy body
	io.Copy(w, resp.Body)
	resp.Body.Close()
}
