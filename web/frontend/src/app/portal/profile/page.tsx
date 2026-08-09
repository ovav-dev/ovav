"use client";

import { useEffect, useState } from "react";
import api, { type SessionUser } from "@/lib/api";

export default function ProfilePage() {
  const [user, setUser] = useState<SessionUser | null>(null);
  const [loading, setLoading] = useState(true);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    api.getSession().then(setUser).catch(console.error).finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="max-w-2xl mx-auto px-6 py-12"><div className="animate-spin text-2xl">⚙️</div></div>;

  return (
    <div className="max-w-2xl mx-auto px-6 py-12">
      <h1 className="text-2xl font-bold mb-8">Tu perfil</h1>
      <div className="space-y-6">
        <div>
          <label className="text-sm text-gray-400 block mb-1">Email</label>
          <input type="email" defaultValue={user?.email || ""} className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-300 text-sm focus:border-emerald-500 focus:outline-none" />
        </div>
        <div>
          <label className="text-sm text-gray-400 block mb-1">Nombre</label>
          <input type="text" defaultValue={user?.name || ""} className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded-lg text-gray-300 text-sm focus:border-emerald-500 focus:outline-none" />
        </div>
        <button onClick={() => { setSaved(true); setTimeout(() => setSaved(false), 3000); }} className="px-6 py-2.5 bg-emerald-500 hover:bg-emerald-400 text-gray-950 rounded-lg font-medium transition">
          {saved ? "✓ Guardado" : "Guardar cambios"}
        </button>
      </div>
    </div>
  );
}
