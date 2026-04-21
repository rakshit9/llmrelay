"use client";
import { useEffect, useState } from "react";
import { api, APIKey, Project } from "@/lib/api";

export default function KeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [form, setForm] = useState({ name: "", project_id: "", rate_limit_rpm: "60", budget_usd: "" });
  const [error, setError] = useState("");

  const load = async () => {
    const [k, p] = await Promise.all([api.keys.list(), api.projects.list()]);
    setKeys(k);
    setProjects(p);
  };

  useEffect(() => { load().catch(() => {}); }, []);

  const create = async () => {
    if (!form.name || !form.project_id) return;
    try {
      const created = await api.keys.create({
        name: form.name,
        project_id: Number(form.project_id),
        rate_limit_rpm: Number(form.rate_limit_rpm),
        budget_usd: form.budget_usd || undefined,
      });
      setNewKey(created.key ?? null);
      setForm({ name: "", project_id: "", rate_limit_rpm: "60", budget_usd: "" });
      await load();
    } catch (e: any) { setError(e.message); }
  };

  const revoke = async (id: number) => {
    await api.keys.revoke(id);
    await load();
  };

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold">API Keys</h1>

      {newKey && (
        <div className="bg-green-900/40 border border-green-700 rounded-lg p-4">
          <p className="text-green-400 text-sm font-medium mb-1">Key created — copy it now, it won't be shown again:</p>
          <code className="text-green-300 text-sm break-all">{newKey}</code>
          <button onClick={() => setNewKey(null)} className="ml-4 text-xs text-green-500 underline">dismiss</button>
        </div>
      )}

      {/* Create form */}
      <div className="bg-gray-900 border border-gray-800 rounded-lg p-5 space-y-4">
        <h2 className="text-sm font-medium text-gray-400">Create New Key</h2>
        {error && <p className="text-red-400 text-sm">{error}</p>}
        <div className="grid grid-cols-2 gap-3">
          <input className="bg-gray-800 rounded px-3 py-2 text-sm text-white placeholder-gray-500" placeholder="Key name" value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
          <select className="bg-gray-800 rounded px-3 py-2 text-sm text-white" value={form.project_id} onChange={e => setForm(f => ({ ...f, project_id: e.target.value }))}>
            <option value="">Select project</option>
            {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>
          <input className="bg-gray-800 rounded px-3 py-2 text-sm text-white placeholder-gray-500" placeholder="Rate limit (req/min)" value={form.rate_limit_rpm} onChange={e => setForm(f => ({ ...f, rate_limit_rpm: e.target.value }))} />
          <input className="bg-gray-800 rounded px-3 py-2 text-sm text-white placeholder-gray-500" placeholder="Budget USD (optional)" value={form.budget_usd} onChange={e => setForm(f => ({ ...f, budget_usd: e.target.value }))} />
        </div>
        <button onClick={create} className="bg-blue-600 hover:bg-blue-500 text-white text-sm px-4 py-2 rounded">Create Key</button>
      </div>

      {/* Keys table */}
      <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead className="border-b border-gray-800">
            <tr className="text-gray-400 text-left">
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Rate Limit</th>
              <th className="px-4 py-3">Budget</th>
              <th className="px-4 py-3">Spent</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            {keys.map(k => (
              <tr key={k.id} className="border-b border-gray-800/50">
                <td className="px-4 py-3 text-white">{k.name}</td>
                <td className="px-4 py-3 text-gray-300">{k.rate_limit_rpm} rpm</td>
                <td className="px-4 py-3 text-gray-300">{k.budget_usd ? `$${k.budget_usd}` : "∞"}</td>
                <td className="px-4 py-3 text-gray-300">${Number(k.spent_usd).toFixed(4)}</td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-0.5 rounded text-xs ${k.is_active ? "bg-green-900 text-green-400" : "bg-gray-800 text-gray-500"}`}>
                    {k.is_active ? "active" : "revoked"}
                  </span>
                </td>
                <td className="px-4 py-3">
                  {k.is_active && <button onClick={() => revoke(k.id)} className="text-xs text-red-400 hover:text-red-300">Revoke</button>}
                </td>
              </tr>
            ))}
            {keys.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-gray-600">No keys yet</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
