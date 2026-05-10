'use client';

import React from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  Edge,
  Node,
  Position,
} from 'reactflow';
import 'reactflow/dist/style.css';

interface TaskNode {
  id: string;
  name: string;
  state: string;
  task_type?: string;
  depends_on?: string;
  retry_count?: number;
  error_message?: string;
}

function parseDependencies(dependsOn?: string): string[] {
  if (!dependsOn) {
    return [];
  }

  try {
    const parsed = JSON.parse(dependsOn);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function levelForTask(taskId: string, tasks: TaskNode[], levelCache: Map<string, number>): number {
  if (levelCache.has(taskId)) {
    return levelCache.get(taskId) || 0;
  }

  const task = tasks.find((item) => item.id === taskId);
  if (!task) {
    return 0;
  }

  const dependencies = parseDependencies(task.depends_on);
  if (!dependencies.length) {
    levelCache.set(taskId, 0);
    return 0;
  }

  const level = 1 + Math.max(...dependencies.map((dependency) => levelForTask(dependency, tasks, levelCache)));
  levelCache.set(taskId, level);
  return level;
}

function colorForState(state: string): string {
  switch (state) {
    case 'completed':
      return '#10b981';
    case 'running':
    case 'assigned':
      return '#3b82f6';
    case 'retrying':
      return '#f59e0b';
    case 'failed':
      return '#ef4444';
    default:
      return '#334155';
  }
}

export function ExecutionGraph({ tasks }: { tasks: TaskNode[] }) {
  if (!tasks.length) {
    return (
      <div className="rounded-xl border border-slate-700 bg-slate-950/70 p-6 text-slate-400">
        No tasks have been scheduled yet.
      </div>
    );
  }

  const levelCache = new Map<string, number>();
  const nodes: Node[] = tasks.map((task, index) => {
    const level = levelForTask(task.id, tasks, levelCache);
    return {
      id: task.id,
      position: {
        x: 60 + level * 260,
        y: 40 + index * 130,
      },
      data: {
        label: (
          <div className="min-w-[190px] rounded-lg border border-slate-600 bg-slate-900 px-3 py-2 text-left text-white shadow-lg">
            <div className="text-xs uppercase tracking-[0.25em] text-slate-400">{task.task_type || 'task'}</div>
            <div className="mt-1 text-sm font-semibold">{task.name}</div>
            <div className="mt-2 text-xs text-slate-300">State: {task.state}</div>
            {task.retry_count ? <div className="text-xs text-slate-300">Retries: {task.retry_count}</div> : null}
            {task.error_message ? <div className="mt-1 text-xs text-red-300">{task.error_message}</div> : null}
          </div>
        ),
      },
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
      style: {
        border: 'none',
        background: 'transparent',
      },
    };
  });

  const edges: Edge[] = tasks.flatMap((task) => {
    const dependencies = parseDependencies(task.depends_on);
    return dependencies.map((dependency) => ({
      id: `${dependency}-${task.id}`,
      source: dependency,
      target: task.id,
      animated: task.state === 'running' || task.state === 'assigned',
      style: {
        stroke: colorForState(task.state),
        strokeWidth: 2,
      },
    }));
  });

  return (
    <div className="h-[680px] rounded-2xl border border-slate-700 bg-slate-950/80 shadow-2xl shadow-slate-950/60">
      <ReactFlow nodes={nodes} edges={edges} fitView nodesDraggable={false} nodesConnectable={false}>
        <MiniMap zoomable pannable nodeColor={(node) => colorForState(String(node.data?.state || 'pending'))} />
        <Controls />
        <Background gap={24} size={1} color="#1f2937" />
      </ReactFlow>
    </div>
  );
}
