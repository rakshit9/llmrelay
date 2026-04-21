"use client";
import { useEffect, useState } from "react";
import { api, Project } from "@/lib/api";

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [name, setName] = useState("");

  const load = () => api.projects.list().then(setProjects).catch(() => {});
  useEffect(() => { load(); }, []);

  const create = async () => {
    if (!name.trim()) return;
    await api.projects.create(name.trim());
    setName("");
    await load();
  };

  const remove = async (id: number) => {
    await api.projects.delete(id);
    await load();
  };

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold">Projects</h1>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-5 flex gap-3">
        <input className="bg-gray-800 rounded px-3 py-2 text-sm text-white placeholder-gray-500 flex-1" placeholder="Project name" value={name} onChange={e => setName(e.target.value)} onKeyDown={e => e.key === "Enter" && create()} />
        <button onClick={create} className="bg-blue-600 hover:bg-blue-500 text-white text-sm px-4 py-2 rounded">Create</button>
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead className="border-b border-gray-800">
            <tr className="text-gray-400 text-left">
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Created</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            {projects.map(p => (
              <tr key={p.id} className="border-b border-gray-800/50">
                <td className="px-4 py-3 text-white">{p.name}</td>
                <td className="px-4 py-3 text-gray-400">{new Date(p.created_at).toLocaleDateString()}</td>
                <td className="px-4 py-3">
                  <button onClick={() => remove(p.id)} className="text-xs text-red-400 hover:text-red-300">Delete</button>
                </td>
              </tr>
            ))}
            {projects.length === 0 && (
              <tr><td colSpan={3} className="px-4 py-6 text-center text-gray-600">No projects yet</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
