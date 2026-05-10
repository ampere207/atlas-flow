import axios, { AxiosInstance } from 'axios';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000';

class APIClient {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add token to requests
    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('access_token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    });
  }

  // Auth endpoints
  async signup(email: string, fullName: string, password: string) {
    const response = await this.client.post('/auth/signup', {
      email,
      full_name: fullName,
      password,
    });
    return response.data;
  }

  async login(email: string, password: string) {
    const response = await this.client.post('/auth/login', {
      email,
      password,
    });
    return response.data;
  }

  async refreshToken(refreshToken: string) {
    const response = await this.client.post('/auth/refresh', {
      refresh_token: refreshToken,
    });
    return response.data;
  }

  // Workflow endpoints
  async createWorkflow(name: string, metadata?: Record<string, any>) {
    const response = await this.client.post('/workflows', {
      name,
      metadata: metadata || {},
    });
    return response.data;
  }

  async getWorkflow(id: string) {
    const response = await this.client.get(`/workflows/${id}`);
    return response.data;
  }

  async listWorkflows(limit = 10, offset = 0) {
    const response = await this.client.get('/workflows', {
      params: { limit, offset },
    });
    return response.data;
  }

  async updateWorkflowStatus(id: string, status: string) {
    const response = await this.client.put(`/workflows/${id}/status`, {
      status,
    });
    return response.data;
  }

  async executeWorkflow(id: string) {
    const response = await this.client.post(`/workflows/${id}/execute`);
    return response.data;
  }

  async cancelWorkflow(id: string) {
    const response = await this.client.post(`/workflows/${id}/cancel`);
    return response.data;
  }

  async getWorkflowExecutionStatus(id: string) {
    const response = await this.client.get(`/workflows/${id}/status`);
    return response.data;
  }

  async listWorkflowTasks(id: string) {
    const response = await this.client.get(`/workflows/${id}/tasks`);
    return response.data;
  }

  async listWorkflowHistory(id: string) {
    const response = await this.client.get(`/workflows/${id}/history`);
    return response.data;
  }

  // Worker endpoints
  async registerWorker(name: string) {
    const response = await this.client.post('/workers', { name });
    return response.data;
  }

  async getWorker(id: string) {
    const response = await this.client.get(`/workers/${id}`);
    return response.data;
  }

  async listWorkers(limit = 10, offset = 0) {
    const response = await this.client.get('/workers', {
      params: { limit, offset },
    });
    return response.data;
  }

  async recordHeartbeat(id: string, status: string) {
    const response = await this.client.post(`/workers/${id}/heartbeat`, {
      status,
    });
    return response.data;
  }
}

export const apiClient = new APIClient();
