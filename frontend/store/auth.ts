import { create } from 'zustand';
import { apiClient } from '@/lib/api-client';

interface User {
  id: string;
  email: string;
  full_name: string;
}

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isLoading: boolean;
  error: string | null;

  signup: (email: string, fullName: string, password: string) => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  setTokens: (accessToken: string, refreshToken: string, user: User) => void;
  refreshAccessToken: () => Promise<void>;
  isAuthenticated: () => boolean;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  accessToken: null,
  refreshToken: null,
  isLoading: false,
  error: null,

  signup: async (email: string, fullName: string, password: string) => {
    set({ isLoading: true, error: null });
    try {
      const result = await apiClient.signup(email, fullName, password);
      localStorage.setItem('access_token', result.data.access_token);
      localStorage.setItem('refresh_token', result.data.refresh_token);
      set({
        user: result.data.user,
        accessToken: result.data.access_token,
        refreshToken: result.data.refresh_token,
        isLoading: false,
      });
    } catch (error: any) {
      set({
        error: error.response?.data?.error || 'Signup failed',
        isLoading: false,
      });
      throw error;
    }
  },

  login: async (email: string, password: string) => {
    set({ isLoading: true, error: null });
    try {
      const result = await apiClient.login(email, password);
      localStorage.setItem('access_token', result.data.access_token);
      localStorage.setItem('refresh_token', result.data.refresh_token);
      set({
        user: result.data.user,
        accessToken: result.data.access_token,
        refreshToken: result.data.refresh_token,
        isLoading: false,
      });
    } catch (error: any) {
      set({
        error: error.response?.data?.error || 'Login failed',
        isLoading: false,
      });
      throw error;
    }
  },

  logout: () => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    set({
      user: null,
      accessToken: null,
      refreshToken: null,
    });
  },

  setTokens: (accessToken: string, refreshToken: string, user: User) => {
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
    set({
      user,
      accessToken,
      refreshToken,
    });
  },

  refreshAccessToken: async () => {
    const { refreshToken } = get();
    if (!refreshToken) {
      get().logout();
      return;
    }

    try {
      const result = await apiClient.refreshToken(refreshToken);
      localStorage.setItem('access_token', result.data.access_token);
      set({
        accessToken: result.data.access_token,
        user: result.data.user,
      });
    } catch (error) {
      get().logout();
      throw error;
    }
  },

  isAuthenticated: () => {
    return get().accessToken !== null;
  },
}));

// Initialize auth state from localStorage
export function initializeAuth() {
  const accessToken = localStorage.getItem('access_token');
  const refreshToken = localStorage.getItem('refresh_token');

  if (accessToken && refreshToken) {
    useAuthStore.setState({
      accessToken,
      refreshToken,
    });
  }
}
