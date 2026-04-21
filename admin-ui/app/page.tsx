"use client";
import { useEffect, useState } from "react";
import { api, UsageStats } from "@/lib/api";

function StatCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
      <p className="text-gray-400 text-sm">{label}</p>
      <p className="text-white text-2xl font-semibold mt-1">{value}</p>
    </div>
  );
}

export default function Dashboard() {
  const [stats, setStats] = useState<UsageStats | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api.analytics.stats().then(setStats).catch((e) => setError(e.message));
  }, []);

  if (error) return <p className="text-red-400">Failed to load stats: {error}</p>;
  if (!stats) return <p className="text-gray-400">Loading...</p>;

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold">Dashboard</h1>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Total Requests" value={stats.total_requests.toLocaleString()} />
        <StatCard label="Total Tokens" value={stats.total_tokens.toLocaleString()} />
        <StatCard label="Total Cost" value={`$${Number(stats.total_cost_usd).toFixed(4)}`} />
        <StatCard label="Cache Hit Rate" value={`${(stats.cache_hit_rate * 100).toFixed(1)}%`} />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
          <h2 className="text-sm font-medium text-gray-400 mb-4">Requests by Model</h2>
          {Object.entries(stats.requests_by_model).map(([model, count]) => (
            <div key={model} className="flex justify-between py-1 text-sm">
              <span className="text-gray-300">{model}</span>
              <span className="text-white font-medium">{count}</span>
            </div>
          ))}
          {Object.keys(stats.requests_by_model).length === 0 && (
            <p className="text-gray-600 text-sm">No data yet</p>
          )}
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
          <h2 className="text-sm font-medium text-gray-400 mb-4">Requests by Provider</h2>
          {Object.entries(stats.requests_by_provider).map(([provider, count]) => (
            <div key={provider} className="flex justify-between py-1 text-sm">
              <span className="text-gray-300">{provider}</span>
              <span className="text-white font-medium">{count}</span>
            </div>
          ))}
          {Object.keys(stats.requests_by_provider).length === 0 && (
            <p className="text-gray-600 text-sm">No data yet</p>
          )}
        </div>
      </div>
    </div>
  );
}
