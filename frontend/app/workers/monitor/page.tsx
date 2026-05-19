'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Sidebar from '@/components/Sidebar';
import { apiClient } from '@/lib/api-client';
import { useAuthStore } from '@/store/auth';
import { Activity, Server, Shield, Zap, Cpu, MemoryStick as Memory, Globe, Clock } from 'lucide-react';

interface Worker {
  worker_id: string;
  status: string;
  capabilities: string[];
  capacity: number;
  running_tasks: number;
  completed_tasks: number;
  failed_tasks: number;
  last_heartbeat: string;
}

export default function WorkerMonitorPage() {
  const router = useRouter();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/auth/login');
      return;
    }

    let active = true;
    const fetchWorkers = async () => {
      try {
        const data = await apiClient.listWorkers(50, 0);
        if (active) {
          setWorkers(data || []);
          setLoading(false);
        }
      } catch (err) {
        console.error('Failed to fetch workers', err);
      }
    };

    fetchWorkers();
    const interval = setInterval(fetchWorkers, 5000);
    return () => {
      active = false;
      clearInterval(interval);
    };
  }, [isAuthenticated, router]);

  const activeCount = workers.filter(w => w.status === 'connected').length;
  const totalTasks = workers.reduce((acc, w) => acc + w.running_tasks, 0);

  return (
    <div className="flex min-h-screen bg-[#050816] text-white">
      <Sidebar />
      <main className="flex-1 ml-64 overflow-auto">
        <div className="relative min-h-screen p-8 bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.15),_transparent_40%)]">
          <header className="mb-10">
            <div className="flex items-center gap-3 text-cyan-400 mb-2">
              <Activity size={20} />
              <span className="text-xs uppercase tracking-[0.4em] font-semibold">Real-time Cluster Fleet</span>
            </div>
            <h1 className="text-4xl font-bold tracking-tight">Worker Fleet Monitoring</h1>
            <p className="mt-2 text-slate-400 max-w-2xl">
              Inspect heartbeats, task distribution, and capability mapping across your distributed worker fleet.
            </p>
          </header>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-10">
            <StatsCard icon={<Server className="text-cyan-400" />} label="Nodes Online" value={activeCount.toString()} sub={`of ${workers.length} registered`} />
            <StatsCard icon={<Zap className="text-amber-400" />} label="Active Tasks" value={totalTasks.toString()} sub="currently executing" />
            <StatsCard icon={<Shield className="text-emerald-400" />} label="Fleet Health" value="98.2%" sub="uptime average" />
            <StatsCard icon={<Clock className="text-violet-400" />} label="Avg Heartbeat" value="2.4s" sub="latency jitter" />
          </div>

          <div className="grid grid-cols-1 gap-6">
            {workers.map(worker => (
              <div key={worker.worker_id} className="group relative overflow-hidden rounded-3xl border border-white/10 bg-white/[0.03] p-6 transition hover:bg-white/[0.05]">
                <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6">
                  <div className="flex items-center gap-5">
                    <div className={`flex h-14 w-14 items-center justify-center rounded-2xl ${worker.status === 'connected' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'}`}>
                      <Cpu size={28} />
                    </div>
                    <div>
                      <div className="flex items-center gap-3">
                        <h3 className="text-xl font-semibold text-white">{worker.worker_id}</h3>
                        <StatusPill status={worker.status} />
                      </div>
                      <div className="mt-2 flex flex-wrap gap-2">
                        {worker.capabilities.map(cap => (
                          <span key={cap} className="rounded-full bg-slate-800 px-2.5 py-0.5 text-[10px] font-medium text-slate-300 border border-white/5">
                            {cap}
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-8">
                    <WorkerMetric icon={<Activity size={14} />} label="Workload" value={`${Math.round((worker.running_tasks / worker.capacity) * 100)}%`} />
                    <WorkerMetric icon={<Zap size={14} />} label="Running" value={worker.running_tasks.toString()} />
                    <WorkerMetric icon={<CheckCircle size={14} />} label="Success" value={worker.completed_tasks.toString()} />
                    <WorkerMetric icon={<Clock size={14} />} label="Last Pulse" value={new Date(worker.last_heartbeat).toLocaleTimeString()} />
                  </div>
                </div>

                {/* Progress Bar */}
                <div className="mt-6 h-1.5 w-full overflow-hidden rounded-full bg-white/5">
                  <div 
                    className="h-full bg-gradient-to-r from-cyan-500 to-blue-500 transition-all duration-1000" 
                    style={{ width: `${(worker.running_tasks / worker.capacity) * 100}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      </main>
    </div>
  );
}

function StatsCard({ icon, label, value, sub }: { icon: React.ReactNode; label: string; value: string; sub: string }) {
  return (
    <div className="rounded-3xl border border-white/10 bg-white/[0.04] p-6 backdrop-blur-xl">
      <div className="flex items-center justify-between mb-4">
        <div className="p-2 rounded-xl bg-white/5">{icon}</div>
      </div>
      <p className="text-sm text-slate-400 font-medium">{label}</p>
      <h3 className="text-3xl font-bold mt-1 text-white">{value}</h3>
      <p className="text-xs text-slate-500 mt-1 uppercase tracking-wider">{sub}</p>
    </div>
  );
}

function WorkerMetric({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-1.5 text-slate-500 mb-1">
        {icon}
        <span className="text-[10px] uppercase tracking-wider font-semibold">{label}</span>
      </div>
      <span className="text-lg font-bold text-slate-200">{value}</span>
    </div>
  );
}

function StatusPill({ status }: { status: string }) {
  const isActive = status === 'connected';
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider border ${isActive ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20' : 'bg-rose-500/10 text-rose-400 border-rose-500/20'}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${isActive ? 'bg-emerald-400 animate-pulse' : 'bg-rose-400'}`} />
      {status}
    </span>
  );
}

function CheckCircle({ size, className }: { size?: number; className?: string }) {
  return (
    <svg 
      width={size || 16} 
      height={size || 16} 
      viewBox="0 0 24 24" 
      fill="none" 
      stroke="currentColor" 
      strokeWidth="2" 
      strokeLinecap="round" 
      strokeLinejoin="round" 
      className={className}
    >
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
      <polyline points="22 4 12 14.01 9 11.01" />
    </svg>
  );
}
