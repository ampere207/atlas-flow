'use client';

import Link from 'next/link';
import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';

import Sidebar from '@/components/Sidebar';
import { apiClient } from '@/lib/api-client';
import { useAuthStore } from '@/store/auth';

interface Workflow {
  id: string;
  name: string;
  status: string;
  created_at: string;
}

interface Worker {
  id: string;
  name: string;
  status: string;
  last_heartbeat: string;
}

const statusPalette: Record<string, string> = {
  pending: 'from-amber-400 to-orange-500',
  running: 'from-cyan-400 to-blue-500',
  completed: 'from-emerald-400 to-green-500',
  failed: 'from-rose-400 to-red-500',
  active: 'from-emerald-400 to-green-500',
  idle: 'from-sky-400 to-cyan-500',
  offline: 'from-slate-500 to-slate-600',
};

export default function DashboardPage() {
  const router = useRouter();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  const [loading, setLoading] = useState(true);
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [workers, setWorkers] = useState<Worker[]>([]);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/auth/login');
      return;
    }

    let active = true;

    const load = async () => {
      setLoading(true);
      try {
        const [workflowResult, workerResult] = await Promise.all([
          apiClient.listWorkflows(8, 0),
          apiClient.listWorkers(20, 0),
        ]);

        if (!active) {
          return;
        }

        setWorkflows(workflowResult.data || []);
        setWorkers(workerResult.data || []);
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error);
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };

    load();
    const interval = setInterval(load, 15000);
    return () => {
      active = false;
      clearInterval(interval);
    };
  }, [isAuthenticated, router]);

  const metrics = useMemo(() => {
    const summary = workflows.reduce(
      (acc, workflow) => {
        acc.total += 1;
        const status = workflow.status.toLowerCase();
        if (status === 'pending') acc.pending += 1;
        else if (status === 'running') acc.running += 1;
        else if (status === 'completed') acc.completed += 1;
        else if (status === 'failed') acc.failed += 1;
        return acc;
      },
      { total: 0, pending: 0, running: 0, completed: 0, failed: 0 }
    );

    const activeWorkers = workers.filter((worker) => worker.status === 'active' || worker.status === 'idle').length;
    const offlineWorkers = Math.max(workers.length - activeWorkers, 0);

    return {
      ...summary,
      activeWorkers,
      offlineWorkers,
    };
  }, [workflows, workers]);

  const recentWorkflows = workflows.slice(0, 6);
  const activeWorkerList = workers.filter((worker) => worker.status === 'active' || worker.status === 'idle').slice(0, 5);

  const workflowSeries = useMemo(() => {
    const days = Array.from({ length: 7 }, (_, index) => {
      const date = new Date();
      date.setDate(date.getDate() - (6 - index));
      return date.toISOString().slice(0, 10);
    });

    return days.map((day) => {
      const count = workflows.filter((workflow) => workflow.created_at?.startsWith(day)).length;
      return {
        day,
        count,
        label: new Date(day).toLocaleDateString(undefined, { weekday: 'short' }),
      };
    });
  }, [workflows]);

  const workerStatusBreakdown = useMemo(() => {
    const active = workers.filter((worker) => worker.status === 'active').length;
    const idle = workers.filter((worker) => worker.status === 'idle').length;
    const offline = workers.filter((worker) => worker.status !== 'active' && worker.status !== 'idle').length;

    return [
      { label: 'Active', value: active, tone: 'emerald' },
      { label: 'Idle', value: idle, tone: 'sky' },
      { label: 'Offline', value: offline, tone: 'rose' },
    ];
  }, [workers]);

  return (
    <div className="flex min-h-screen bg-[#050816] text-white">
      <Sidebar />
      <main className="flex-1 ml-64 overflow-auto">
        <div className="relative min-h-screen overflow-hidden bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.22),_transparent_26%),radial-gradient(circle_at_80%_10%,_rgba(16,185,129,0.14),_transparent_24%),radial-gradient(circle_at_bottom_right,_rgba(124,58,237,0.16),_transparent_22%),linear-gradient(180deg,_#07111e_0%,_#050816_44%,_#040711_100%)]">
          <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(255,255,255,0.04)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.04)_1px,transparent_1px)] bg-[size:72px_72px] opacity-[0.08]" />
          <div className="pointer-events-none absolute left-12 top-28 h-72 w-72 rounded-full bg-cyan-500/12 blur-3xl" />
          <div className="pointer-events-none absolute right-0 top-1/2 h-80 w-80 rounded-full bg-violet-500/10 blur-3xl" />

          <header className="sticky top-0 z-20 border-b border-white/10 bg-[#07111d]/70 backdrop-blur-xl">
            <div className="flex items-center justify-between gap-6 px-8 py-6">
              <div>
                <p className="text-[10px] uppercase tracking-[0.45em] text-cyan-300/70">AtlasFlow Control Plane</p>
                <h1 className="mt-2 text-3xl font-semibold text-white md:text-4xl">Executive operations dashboard</h1>
                <p className="mt-1 max-w-2xl text-sm text-slate-400 md:text-base">A product-grade command center for workflow health, worker capacity, and execution velocity.</p>
              </div>
              <div className="hidden gap-3 xl:flex">
                <Pill label="Gateway" value="8000" tone="emerald" />
                <Pill label="Workers" value={`${metrics.activeWorkers}/${workers.length || 0}`} tone="sky" />
              </div>
            </div>
          </header>

          <div className="relative z-10 px-6 py-8 md:px-8 lg:px-10 space-y-8">
            <section className="grid gap-6 xl:grid-cols-[1.45fr_0.95fr]">
              <div className="relative overflow-hidden rounded-[2rem] border border-white/10 bg-white/[0.06] p-8 shadow-[0_30px_100px_rgba(0,0,0,0.35)] backdrop-blur-2xl">
                <div className="absolute inset-0 bg-[linear-gradient(135deg,rgba(34,211,238,0.12),transparent_28%,rgba(99,102,241,0.12)_72%,transparent_100%)]" />
                <div className="relative flex flex-col gap-8 lg:flex-row lg:items-end lg:justify-between">
                  <div className="max-w-2xl space-y-5">
                    <div className="flex flex-wrap gap-2">
                      <Badge tone="cyan">Live system</Badge>
                      <Badge tone="violet">Orchestration</Badge>
                      <Badge tone="emerald">Monitoring</Badge>
                    </div>
                    <h2 className="max-w-2xl text-4xl font-semibold tracking-tight text-white md:text-6xl">
                      See workflows and workers like a real operations product.
                    </h2>
                    <p className="max-w-2xl text-sm leading-7 text-slate-300 md:text-base">
                      This surface is designed like a control room: denser hierarchy, calmer spacing, stronger contrast, richer cards, and live trend visuals that feel closer to Google Analytics than a generic admin page.
                    </p>

                    <div className="grid max-w-xl gap-3 sm:grid-cols-3">
                      <MiniMetric label="Workflow throughput" value={`${metrics.running + metrics.completed}`} detail="tasks moving" />
                      <MiniMetric label="Worker health" value={`${metrics.activeWorkers}`} detail="nodes online" />
                      <MiniMetric label="Exceptions" value={`${metrics.failed}`} detail="needs attention" />
                    </div>
                  </div>

                  <div className="grid min-w-[300px] grid-cols-2 gap-3">
                    <ActionCard href="/workflows" title="Create workflow" subtitle="Design a new DAG" tone="blue" />
                    <ActionCard href="/workers" title="Register worker" subtitle="Add capacity" tone="violet" />
                    <ActionCard href="/workers/cluster" title="View cluster" subtitle="Inspect nodes" tone="emerald" />
                    <ActionCard href="/timeline" title="Open timeline" subtitle="Review transitions" tone="amber" />
                  </div>
                </div>
              </div>

              <div className="grid gap-4 rounded-[2rem] border border-white/10 bg-white/[0.06] p-6 shadow-[0_30px_100px_rgba(0,0,0,0.35)] backdrop-blur-2xl">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-[10px] uppercase tracking-[0.35em] text-slate-400">System pulse</p>
                    <h3 className="mt-2 text-xl font-semibold text-white">At a glance</h3>
                  </div>
                  <span className="rounded-full border border-emerald-400/20 bg-emerald-400/10 px-3 py-1 text-xs text-emerald-200">
                    {loading ? 'Refreshing' : 'Live'}
                  </span>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <StatTile label="Workflows" value={metrics.total} accent="blue" />
                  <StatTile label="Active workers" value={metrics.activeWorkers} accent="emerald" />
                  <StatTile label="Pending" value={metrics.pending} accent="amber" />
                  <StatTile label="Offline" value={metrics.offlineWorkers} accent="rose" />
                </div>

                <div className="rounded-3xl border border-white/10 bg-slate-950/40 p-5">
                  <div className="flex items-center justify-between text-sm text-slate-300">
                    <span>Execution readiness</span>
                    <span>{metrics.running > 0 ? 'Healthy' : 'Idle'}</span>
                  </div>
                  <div className="mt-4 h-3 overflow-hidden rounded-full bg-white/5">
                    <div className="h-full w-[78%] rounded-full bg-gradient-to-r from-cyan-400 via-blue-500 to-emerald-400 shadow-[0_0_24px_rgba(34,211,238,0.35)]" />
                  </div>
                  <p className="mt-3 text-xs leading-6 text-slate-400">
                    Routing, worker discovery, and execution streaming are wired through the gateway.
                  </p>
                </div>

                <div className="grid gap-3">
                  {workerStatusBreakdown.map((item) => (
                    <div key={item.label} className="rounded-2xl border border-white/10 bg-slate-950/35 p-4">
                      <div className="flex items-center justify-between text-sm text-slate-300">
                        <span>{item.label}</span>
                        <span>{item.value}</span>
                      </div>
                      <div className="mt-3 h-2 overflow-hidden rounded-full bg-white/5">
                        <div className={`h-full rounded-full bg-gradient-to-r ${statusPalette[item.label.toLowerCase()] || 'from-slate-400 to-slate-500'}`} style={{ width: `${Math.max(15, item.value * 20)}%` }} />
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </section>

            <section className="grid gap-6 xl:grid-cols-[1.35fr_0.95fr]">
              <Panel title="Workflow performance" subtitle="7-day creation trend and current mix">
                <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
                  <div className="rounded-[1.5rem] border border-white/10 bg-slate-950/40 p-5">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Trend chart</p>
                        <h4 className="mt-2 text-lg font-semibold text-white">Workflow intake</h4>
                      </div>
                      <span className="rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-xs text-cyan-200">7 days</span>
                    </div>
                    <div className="mt-6">
                      <BarChart data={workflowSeries} />
                    </div>
                  </div>

                  <div className="grid gap-3">
                    {[
                      { label: 'Pending', value: metrics.pending, tone: 'amber' },
                      { label: 'Running', value: metrics.running, tone: 'cyan' },
                      { label: 'Completed', value: metrics.completed, tone: 'emerald' },
                      { label: 'Failed', value: metrics.failed, tone: 'rose' },
                    ].map((item) => (
                      <div key={item.label} className="rounded-[1.2rem] border border-white/10 bg-slate-950/40 p-4">
                        <div className="flex items-center justify-between">
                          <p className="text-sm text-slate-300">{item.label}</p>
                          <p className="text-xl font-semibold text-white">{item.value}</p>
                        </div>
                        <div className="mt-3 h-2 rounded-full bg-white/5">
                          <div
                            className={`h-full rounded-full bg-gradient-to-r ${
                              item.tone === 'amber'
                                ? 'from-amber-400 to-orange-500'
                                : item.tone === 'cyan'
                                  ? 'from-cyan-400 to-blue-500'
                                  : item.tone === 'emerald'
                                    ? 'from-emerald-400 to-green-500'
                                    : 'from-rose-400 to-red-500'
                            }`}
                            style={{ width: `${Math.min(100, 20 + item.value * 18)}%` }}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </Panel>

              <Panel title="Worker status" subtitle="Cluster health and heartbeat summary">
                <div className="space-y-4">
                  {activeWorkerList.length === 0 ? (
                    <EmptyState title="No active workers" description="Register a worker to start serving tasks." actionHref="/workers" actionLabel="Register worker" compact />
                  ) : (
                    activeWorkerList.map((worker) => (
                      <div key={worker.id} className="rounded-[1.1rem] border border-white/10 bg-slate-950/40 p-4 transition hover:border-cyan-400/25">
                        <div className="flex items-center justify-between gap-3">
                          <div className="min-w-0">
                            <p className="truncate font-medium text-white">{worker.name}</p>
                            <p className="mt-1 text-xs text-slate-400">Heartbeat {new Date(worker.last_heartbeat).toLocaleTimeString()}</p>
                          </div>
                          <StatusBadge status={worker.status} />
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </Panel>
            </section>

            <section className="grid gap-6 xl:grid-cols-[1.35fr_0.95fr]">
              <Panel title="Recent workflows" subtitle="Latest orchestration items and their current state">
                <div className="space-y-3">
                  {loading ? (
                    <div className="flex items-center justify-center py-12">
                      <div className="h-10 w-10 animate-spin rounded-full border-2 border-cyan-400 border-t-transparent" />
                    </div>
                  ) : recentWorkflows.length === 0 ? (
                    <EmptyState title="No workflows yet" description="Create your first workflow to populate the dashboard feed." actionHref="/workflows" actionLabel="Create workflow" compact />
                  ) : (
                    recentWorkflows.map((workflow, index) => (
                      <Link
                        key={workflow.id}
                        href={`/workflows/${workflow.id}`}
                        className="group flex items-center justify-between gap-4 rounded-[1.1rem] border border-white/10 bg-slate-950/40 px-4 py-4 transition hover:border-cyan-400/30 hover:bg-slate-950/70"
                      >
                        <div className="flex items-center gap-4 min-w-0">
                          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-cyan-400/20 to-blue-500/20 text-sm font-semibold text-cyan-100">
                            #{index + 1}
                          </div>
                          <div className="min-w-0">
                            <p className="truncate font-medium text-white group-hover:text-cyan-200">{workflow.name}</p>
                            <p className="mt-1 text-xs text-slate-400">Created {new Date(workflow.created_at).toLocaleString()}</p>
                          </div>
                        </div>
                        <StatusBadge status={workflow.status} />
                      </Link>
                    ))
                  )}
                </div>
              </Panel>

              <Panel title="Quick actions" subtitle="Shortcuts into the main operational flows">
                <div className="grid gap-3">
                  <QuickAction href="/workflows" title="Create workflow" desc="Start a new orchestration run" accent="blue" />
                  <QuickAction href="/workers" title="Register worker" desc="Add execution capacity" accent="violet" />
                  <QuickAction href="/workers/cluster" title="Monitor cluster" desc="Inspect live worker state" accent="emerald" />
                  <QuickAction href="/timeline" title="Open timeline" desc="Review execution transitions" accent="amber" />
                </div>
              </Panel>
            </section>
          </div>
        </div>
      </main>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const key = status.toLowerCase();
  const classes =
    key === 'running'
      ? 'bg-blue-500/15 text-blue-200 border-blue-400/20'
      : key === 'completed'
        ? 'bg-emerald-500/15 text-emerald-200 border-emerald-400/20'
        : key === 'pending'
          ? 'bg-amber-500/15 text-amber-200 border-amber-400/20'
          : key === 'active' || key === 'idle'
            ? 'bg-cyan-500/15 text-cyan-200 border-cyan-400/20'
            : 'bg-rose-500/15 text-rose-200 border-rose-400/20';

  return <span className={`inline-flex shrink-0 items-center rounded-full border px-3 py-1 text-xs font-medium capitalize ${classes}`}>{status}</span>;
}

function Pill({ label, value, tone }: { label: string; value: string; tone: 'emerald' | 'sky' }) {
  const classes = tone === 'emerald' ? 'bg-emerald-500/10 text-emerald-200 border-emerald-400/20' : 'bg-sky-500/10 text-sky-200 border-sky-400/20';

  return (
    <div className={`rounded-full border px-4 py-2 ${classes}`}>
      <p className="text-[10px] uppercase tracking-[0.25em] opacity-80">{label}</p>
      <p className="text-sm font-semibold">{value}</p>
    </div>
  );
}

function Badge({ children, tone }: { children: string; tone: 'cyan' | 'violet' | 'emerald' }) {
  const classes =
    tone === 'cyan'
      ? 'border-cyan-400/20 bg-cyan-400/10 text-cyan-200'
      : tone === 'violet'
        ? 'border-violet-400/20 bg-violet-400/10 text-violet-200'
        : 'border-emerald-400/20 bg-emerald-400/10 text-emerald-200';

  return <span className={`inline-flex rounded-full border px-3 py-1 text-[11px] font-medium ${classes}`}>{children}</span>;
}

function MiniMetric({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-slate-950/35 p-4">
      <p className="text-[10px] uppercase tracking-[0.28em] text-slate-400">{label}</p>
      <p className="mt-2 text-2xl font-semibold text-white">{value}</p>
      <p className="mt-1 text-xs text-slate-400">{detail}</p>
    </div>
  );
}

function EmptyState({
  title,
  description,
  actionHref,
  actionLabel,
  compact = false,
}: {
  title: string;
  description: string;
  actionHref: string;
  actionLabel: string;
  compact?: boolean;
}) {
  return (
    <div className={`rounded-[1.25rem] border border-dashed border-white/10 bg-slate-950/35 text-center ${compact ? 'px-5 py-6' : 'px-6 py-10'}`}>
      <p className="text-sm font-medium text-white">{title}</p>
      <p className="mt-2 text-sm leading-6 text-slate-400">{description}</p>
      <Link href={actionHref} className="mt-4 inline-flex rounded-full border border-cyan-400/20 bg-cyan-400/10 px-4 py-2 text-sm font-medium text-cyan-200 transition hover:bg-cyan-400/20">
        {actionLabel}
      </Link>
    </div>
  );
}

function StatTile({ label, value, accent }: { label: string; value: number; accent: 'blue' | 'emerald' | 'amber' | 'rose' }) {
  const tone =
    accent === 'blue'
      ? 'from-blue-500/20 to-cyan-500/10 text-cyan-100'
      : accent === 'emerald'
        ? 'from-emerald-500/20 to-green-500/10 text-emerald-100'
        : accent === 'amber'
          ? 'from-amber-500/20 to-orange-500/10 text-amber-100'
          : 'from-rose-500/20 to-red-500/10 text-rose-100';

  return (
    <div className={`rounded-2xl border border-white/10 bg-gradient-to-br ${tone} p-4`}>
      <p className="text-[10px] uppercase tracking-[0.28em] text-white/70">{label}</p>
      <p className="mt-2 text-3xl font-semibold text-white">{value}</p>
    </div>
  );
}

function ActionCard({ href, title, subtitle, tone }: { href: string; title: string; subtitle: string; tone: 'blue' | 'violet' | 'emerald' | 'amber' }) {
  const classes =
    tone === 'blue'
      ? 'from-blue-500/20 to-cyan-500/10 border-blue-400/20 hover:border-blue-300/30'
      : tone === 'violet'
        ? 'from-violet-500/20 to-fuchsia-500/10 border-violet-400/20 hover:border-violet-300/30'
        : tone === 'emerald'
          ? 'from-emerald-500/20 to-green-500/10 border-emerald-400/20 hover:border-emerald-300/30'
          : 'from-amber-500/20 to-orange-500/10 border-amber-400/20 hover:border-amber-300/30';

  return (
    <Link
      href={href}
      className={`group rounded-[1.25rem] border bg-gradient-to-br p-4 transition hover:-translate-y-0.5 hover:shadow-[0_18px_50px_rgba(0,0,0,0.24)] ${classes}`}
    >
      <div className="flex h-full flex-col justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-white group-hover:text-cyan-100">{title}</p>
          <p className="mt-1 text-xs text-slate-300/75">{subtitle}</p>
        </div>
        <div className="flex items-center justify-end text-sm text-slate-200/70 transition group-hover:translate-x-0.5">→</div>
      </div>
    </Link>
  );
}

function Panel({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <section className="overflow-hidden rounded-[2rem] border border-white/10 bg-white/[0.06] shadow-[0_30px_100px_rgba(0,0,0,0.28)] backdrop-blur-2xl">
      <div className="flex items-center justify-between border-b border-white/10 px-6 py-5">
        <div>
          <p className="text-[10px] uppercase tracking-[0.35em] text-slate-400">{subtitle}</p>
          <h3 className="mt-2 text-xl font-semibold text-white">{title}</h3>
        </div>
      </div>
      <div className="p-6">{children}</div>
    </section>
  );
}

function QuickAction({ href, title, desc, accent }: { href: string; title: string; desc: string; accent: 'blue' | 'violet' | 'emerald' | 'amber' }) {
  const tone =
    accent === 'blue'
      ? 'from-blue-500/15 to-cyan-500/10 border-blue-400/20'
      : accent === 'violet'
        ? 'from-violet-500/15 to-fuchsia-500/10 border-violet-400/20'
        : accent === 'emerald'
          ? 'from-emerald-500/15 to-green-500/10 border-emerald-400/20'
          : 'from-amber-500/15 to-orange-500/10 border-amber-400/20';

  return (
    <Link href={href} className={`group rounded-[1.2rem] border bg-gradient-to-br ${tone} p-5 transition hover:-translate-y-0.5 hover:shadow-lg`}>
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="font-medium text-white group-hover:text-cyan-100">{title}</p>
          <p className="mt-1 text-sm text-slate-300/75">{desc}</p>
        </div>
        <span className="text-slate-200/70 transition group-hover:translate-x-0.5">→</span>
      </div>
    </Link>
  );
}

function BarChart({ data }: { data: { label: string; count: number }[] }) {
  const max = Math.max(...data.map((entry) => entry.count), 1);

  return (
    <div className="flex h-[280px] items-end gap-3 rounded-[1.3rem] border border-white/10 bg-slate-950/30 px-5 pb-5 pt-8">
      {data.map((entry) => {
        const height = Math.max(18, (entry.count / max) * 100);
        return (
          <div key={entry.label} className="flex flex-1 flex-col items-center gap-3">
            <div className="flex w-full flex-1 items-end justify-center">
              <div className="w-full max-w-[52px] rounded-t-2xl bg-white/5 p-1">
                <div className="relative rounded-t-[14px] bg-gradient-to-t from-cyan-500 via-blue-500 to-emerald-400 shadow-[0_0_24px_rgba(34,211,238,0.16)]" style={{ height: `${height}%` }}>
                  <div className="absolute inset-x-0 top-0 h-5 rounded-t-[14px] bg-white/15" />
                </div>
              </div>
            </div>
            <div className="text-center">
              <p className="text-sm font-medium text-white">{entry.count}</p>
              <p className="text-[11px] uppercase tracking-[0.25em] text-slate-500">{entry.label}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
