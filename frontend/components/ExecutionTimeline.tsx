import React, { useEffect, useState } from 'react';
import { ExecutionEvent } from '@/hooks/useExecutionStream';

interface ExecutionTimelineProps {
  events: ExecutionEvent[];
  workflowId: string;
  isLive?: boolean;
}

const eventTypeColors: Record<string, string> = {
  workflow_started: 'bg-blue-500/10 border-blue-500/20 text-blue-600',
  task_assigned: 'bg-purple-500/10 border-purple-500/20 text-purple-600',
  task_started: 'bg-cyan-500/10 border-cyan-500/20 text-cyan-600',
  task_completed: 'bg-green-500/10 border-green-500/20 text-green-600',
  task_failed: 'bg-red-500/10 border-red-500/20 text-red-600',
  task_retrying: 'bg-yellow-500/10 border-yellow-500/20 text-yellow-600',
  workflow_completed: 'bg-emerald-500/10 border-emerald-500/20 text-emerald-600',
  workflow_failed: 'bg-rose-500/10 border-rose-500/20 text-rose-600',
};

const eventTypeIcons: Record<string, string> = {
  workflow_started: '▶',
  task_assigned: '📌',
  task_started: '⚡',
  task_completed: '✓',
  task_failed: '✗',
  task_retrying: '🔄',
  workflow_completed: '🏁',
  workflow_failed: '❌',
};

export function ExecutionTimeline({ events, workflowId, isLive = false }: ExecutionTimelineProps) {
  const [displayEvents, setDisplayEvents] = useState<ExecutionEvent[]>([]);

  useEffect(() => {
    setDisplayEvents([...events].reverse()); // Show newest first
  }, [events]);

  if (displayEvents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 px-4">
        <div className="text-gray-400 mb-2">
          {isLive ? '⏳ Waiting for execution events...' : '📭 No events yet'}
        </div>
        {isLive && (
          <div className="flex gap-1">
            <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce"></div>
            <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
            <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-2 max-h-96 overflow-y-auto pr-2">
      {displayEvents.map((event, idx) => (
        <div key={event.event_id || idx} className="flex gap-3 text-sm">
          {/* Timeline connector */}
          <div className="flex flex-col items-center">
            <div
              className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${
                eventTypeColors[event.event_type] || 'bg-gray-500/10 border-gray-500/20 text-gray-600'
              } border`}
            >
              {eventTypeIcons[event.event_type] || '•'}
            </div>
            {idx < displayEvents.length - 1 && (
              <div className="w-0.5 h-6 bg-gray-700/50 my-1"></div>
            )}
          </div>

          {/* Event details */}
          <div className="flex-1 pt-1">
            <div className="flex items-start justify-between">
              <div>
                <p className="font-medium text-gray-200 capitalize">
                  {event.event_type.replace(/_/g, ' ')}
                </p>
                {event.task_id && (
                  <p className="text-xs text-gray-400 mt-0.5">
                    Task: {event.task_id.substring(0, 8)}...
                  </p>
                )}
                {event.worker_id && (
                  <p className="text-xs text-gray-400">
                    Worker: {event.worker_id}
                  </p>
                )}
                {event.data?.duration_ms && (
                  <p className="text-xs text-gray-400">
                    Duration: {event.data.duration_ms}ms
                  </p>
                )}
                {event.error_message && (
                  <p className="text-xs text-red-400 mt-1">
                    Error: {event.error_message}
                  </p>
                )}
              </div>
              <div className="text-right text-xs text-gray-500">
                {new Date(event.timestamp).toLocaleTimeString()}
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
