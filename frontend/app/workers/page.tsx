'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/auth';
import { apiClient } from '@/lib/api-client';
import Sidebar from '@/components/Sidebar';

interface Worker {
  id: string;
  name: string;
  status: string;
  last_heartbeat: string;
}

export default function WorkersPage() {
  const router = useRouter();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [loading, setLoading] = useState(true);
  const [newWorkerName, setNewWorkerName] = useState('');
  const [registering, setRegistering] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/auth/login');
      return;
    }
    fetchWorkers();
  }, [isAuthenticated, router]);

  const fetchWorkers = async () => {
    try {
      setLoading(true);
      const result = await apiClient.listWorkers(100, 0);
      setWorkers(result.data || []);
    } catch (error) {
      console.error('Failed to fetch workers:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleRegisterWorker = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWorkerName.trim()) return;

    try {
      setRegistering(true);
      await apiClient.registerWorker(newWorkerName);
      setNewWorkerName('');
      await fetchWorkers();
    } catch (error) {
      console.error('Failed to register worker:', error);
    } finally {
      setRegistering(false);
    }
  };

  const activeWorkers = workers.filter(
    (w) => w.status === 'active' || w.status === 'idle'
  );
  const inactiveWorkers = workers.filter(
    (w) => w.status !== 'active' && w.status !== 'idle'
  );

  return (
    <div className="flex h-screen bg-slate-950">
      <Sidebar />
      <div className="flex-1 ml-64 overflow-auto">
        <div className="bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 min-h-screen">
          <div className="border-b border-slate-700 sticky top-0 z-10">
            <div className="px-8 py-6 bg-slate-900/50 backdrop-blur flex items-center justify-between">
              <div>
                <h1 className="text-3xl font-bold text-white">Workers</h1>
                <p className="text-slate-400 text-sm mt-1">
                  Register and manage distributed workers
                </p>
              </div>
              <a
                href="/workers/cluster"
                className="px-4 py-2 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
              >
                View Cluster →
              </a>
            </div>
          </div>

          <div className="p-8 space-y-6">
            <div className="bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl p-6">
              <h2 className="text-lg font-semibold text-white mb-4">
                Register New Worker
              </h2>
              <form onSubmit={handleRegisterWorker} className="flex gap-3">
                <input
                  type="text"
                  value={newWorkerName}
                  onChange={(e) => setNewWorkerName(e.target.value)}
                  placeholder="Worker name..."
                  className="flex-1 px-4 py-2 bg-slate-700/50 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-blue-500"
                />
                <button
                  type="submit"
                  disabled={registering || !newWorkerName.trim()}
                  className="px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white rounded-lg font-medium transition-colors"
                >
                  {registering ? 'Registering...' : 'Register'}
                </button>
              </form>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <StatCard label="Total Workers" value={workers.length} />
              <StatCard
                label="Active"
                value={activeWorkers.length}
                color="text-green-400"
              />
              <StatCard
                label="Inactive"
                value={inactiveWorkers.length}
                color="text-slate-400"
              />
            </div>

            <div className="bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl overflow-hidden">
              <div className="px-6 py-4 border-b border-slate-700">
                <h2 className="text-lg font-semibold text-white">
                  Registered Workers
                </h2>
              </div>
              <div className="p-6">
                {loading ? (
                  <div className="flex items-center justify-center py-12">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
                  </div>
                ) : workers.length === 0 ? (
                  <div className="text-center py-12">
                    <p className="text-slate-400 text-lg">
                      No workers registered
                    </p>
                    <p className="text-slate-500 text-sm mt-1">
                      Register your first worker above
                    </p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {workers.map((worker) => (
                      <div
                        key={worker.id}
                        className="p-4 bg-slate-700/30 hover:bg-slate-600/40 border border-slate-600/30 hover:border-slate-500/50 rounded-lg transition-all"
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <p className="text-white font-semibold">
                              {worker.name}
                            </p>
                            <p className="text-xs text-slate-400 mt-1">
                              ID: {worker.id}
                            </p>
                          </div>
                          <span
                            className={`px-3 py-1 rounded-full text-xs font-medium whitespace-nowrap ml-4 ${
                              worker.status === 'active'
                                ? 'bg-green-500/20 text-green-300'
                                : worker.status === 'idle'
                                ? 'bg-yellow-500/20 text-yellow-300'
                                : 'bg-red-500/20 text-red-300'
                            }`}
                          >
                            {worker.status}
                          </span>
                        </div>
                        <p className="text-xs text-slate-400 mt-2">
                          Last heartbeat:{' '}
                          {new Date(worker.last_heartbeat).toLocaleString()}
                        </p>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function StatCard({
  label,
  value,
  color = 'text-white',
}: {
  label: string;
  value: number;
  color?: string;
}) {
  return (
    <div className="bg-slate-800/50 backdrop-blur border border-slate-700 rounded-xl p-6">
      <p className="text-slate-400 text-xs font-medium uppercase tracking-wider">
        {label}
      </p>
      <p className={`text-3xl font-bold mt-2 ${color}`}>{value}</p>
    </div>
  );
}
