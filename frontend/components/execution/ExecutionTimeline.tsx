'use client';

import { format } from 'date-fns';
import { CheckCircle2, Clock, PlayCircle, AlertCircle, RefreshCcw, User } from 'lucide-react';

interface TimelineEvent {
  id: string;
  entity_type: string;
  from_state: string;
  to_state: string;
  reason?: string;
  created_at: string;
  worker_id?: string;
}

export function ExecutionTimeline({ events }: { events: TimelineEvent[] }) {
  if (events.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-2xl border border-slate-800 bg-slate-900/50 py-12 text-slate-500">
        <Clock className="mb-3 h-8 w-8 opacity-20" />
        <p className="text-sm">Waiting for execution events...</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {events.map((event, index) => (
        <div key={event.id} className="relative pl-8">
          {/* Connector Line */}
          {index !== events.length - 1 && (
            <div className="absolute left-[11px] top-6 h-full w-[2px] bg-slate-800" />
          )}

          {/* Icon */}
          <div className="absolute left-0 top-1">
            <EventIcon state={event.to_state} />
          </div>

          <div className="rounded-xl border border-slate-800 bg-slate-900/60 p-4 transition-all hover:bg-slate-900/80">
            <div className="flex items-start justify-between">
              <div>
                <h4 className="font-medium text-slate-200 capitalize">
                  {event.entity_type === 'task' ? 'Task Transition' : 'Workflow Transition'}
                </h4>
                <div className="mt-1 flex items-center gap-2 text-xs font-mono text-slate-500">
                  <span className="rounded bg-slate-800 px-1.5 py-0.5">{event.from_state}</span>
                  <span>→</span>
                  <span className="rounded bg-slate-700 px-1.5 py-0.5 text-slate-200">{event.to_state}</span>
                </div>
              </div>
              <time className="text-[10px] text-slate-500">
                {format(new Date(event.created_at), 'HH:mm:ss.SSS')}
              </time>
            </div>

            {event.reason && (
              <p className="mt-2 text-sm text-slate-400 italic">"{event.reason}"</p>
            )}

            {event.worker_id && (
              <div className="mt-3 flex items-center gap-2 text-xs text-cyan-400/80">
                <User size={12} />
                <span>Worker: {event.worker_id}</span>
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

function EventIcon({ state }: { state: string }) {
  switch (state) {
    case 'completed':
      return <CheckCircle2 size={24} className="text-emerald-500" />;
    case 'running':
    case 'started':
      return <PlayCircle size={24} className="text-cyan-500 animate-pulse" />;
    case 'failed':
      return <AlertCircle size={24} className="text-rose-500" />;
    case 'retrying':
      return <RefreshCcw size={24} className="text-amber-500 animate-spin-slow" />;
    case 'assigned':
      return <Clock size={24} className="text-indigo-500" />;
    default:
      return <Clock size={24} className="text-slate-600" />;
  }
}
