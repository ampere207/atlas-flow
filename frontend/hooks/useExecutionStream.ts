import { useEffect, useState, useCallback, useRef } from 'react';

export interface ExecutionEvent {
  event_id: string;
  event_type: string;
  workflow_id: string;
  task_id?: string;
  worker_id?: string;
  user_id: string;
  timestamp: string;
  data?: Record<string, any>;
  error_message?: string;
  metadata?: Record<string, any>;
}

interface UseExecutionStreamOptions {
  workflowId: string;
  onEvent?: (event: ExecutionEvent) => void;
  onError?: (error: Error) => void;
  autoConnect?: boolean;
}

/**
 * Hook for streaming workflow execution events via Server-Sent Events (SSE)
 * Provides real-time updates as tasks are executed, failed, retried, etc.
 */
export function useExecutionStream({
  workflowId,
  onEvent,
  onError,
  autoConnect = true,
}: UseExecutionStreamOptions) {
  const [isConnected, setIsConnected] = useState(false);
  const [events, setEvents] = useState<ExecutionEvent[]>([]);
  const eventSourceRef = useRef<EventSource | null>(null);

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      return; // Already connected
    }

    try {
      const token = localStorage.getItem('auth_token');
      if (!token) {
        throw new Error('No authentication token found');
      }

      const url = `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000'}/workflows/${workflowId}/stream`;
      const eventSource = new EventSource(url, {
        withCredentials: true,
        headers: {
          Authorization: `Bearer ${token}`,
        },
      } as any);

      eventSource.addEventListener('workflow_started', (event) => {
        const data = JSON.parse(event.data);
        setEvents((prev) => [...prev, data]);
        onEvent?.(data);
      });

      eventSource.addEventListener('task_assigned', (event) => {
        const data = JSON.parse(event.data);
        setEvents((prev) => [...prev, data]);
        onEvent?.(data);
      });

      eventSource.addEventListener('task_started', (event) => {
        const data = JSON.parse(event.data);
        setEvents((prev) => [...prev, data]);
        onEvent?.(data);
      });

      eventSource.addEventListener('task_completed', (event) => {
        const data = JSON.parse(event.data);
        setEvents((prev) => [...prev, data]);
        onEvent?.(data);
      });

      eventSource.addEventListener('task_failed', (event) => {
        const data = JSON.parse(event.data);
        setEvents((prev) => [...prev, data]);
        onEvent?.(data);
      });

      eventSource.addEventListener('task_retrying', (event) => {
        const data = JSON.parse(event.data);
        setEvents((prev) => [...prev, data]);
        onEvent?.(data);
      });

      eventSource.addEventListener('workflow_completed', (event) => {
        const data = JSON.parse(event.data);
        setEvents((prev) => [...prev, data]);
        onEvent?.(data);
        eventSource.close();
        setIsConnected(false);
      });

      eventSource.addEventListener('workflow_failed', (event) => {
        const data = JSON.parse(event.data);
        setEvents((prev) => [...prev, data]);
        onEvent?.(data);
        eventSource.close();
        setIsConnected(false);
      });

      eventSource.onerror = () => {
        const error = new Error('EventSource connection error');
        onError?.(error);
        eventSource.close();
        setIsConnected(false);
      };

      eventSourceRef.current = eventSource;
      setIsConnected(true);
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));
      onError?.(err);
      setIsConnected(false);
    }
  }, [workflowId, onEvent, onError]);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
      setIsConnected(false);
    }
  }, []);

  const clearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  useEffect(() => {
    if (autoConnect) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [workflowId, autoConnect, connect, disconnect]);

  return {
    isConnected,
    events,
    connect,
    disconnect,
    clearEvents,
  };
}
