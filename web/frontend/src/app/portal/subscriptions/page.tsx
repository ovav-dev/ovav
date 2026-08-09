"use client";

import { useState } from "react";

export default function SubscriptionsPage() {
  const [plan] = useState("Pro Anual");
  const [status] = useState("active");

  return (
    <div className="max-w-2xl mx-auto px-6 py-12">
      <h1 className="text-2xl font-bold mb-8">Tus suscripciones</h1>

      <div className="p-6 border border-emerald-500/30 bg-emerald-500/5 rounded-xl mb-8">
        <div className="flex justify-between items-start mb-4">
          <div>
            <h2 className="text-lg font-semibold">{plan}</h2>
            <p className="text-sm text-gray-400">$15/mes (facturación anual)</p>
          </div>
          <span className="px-3 py-1 bg-emerald-500/20 text-emerald-400 text-sm rounded-full capitalize">{status}</span>
        </div>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div><span className="text-gray-500">Inicio</span><p className="text-gray-300">15 Ene 2026</p></div>
          <div><span className="text-gray-500">Próximo cobro</span><p className="text-gray-300">15 Ene 2027</p></div>
          <div><span className="text-gray-500">Método</span><p className="text-gray-300">Visa ••••4242</p></div>
          <div><span className="text-gray-500">Instancias</span><p className="text-gray-300">1 de 3</p></div>
        </div>
        <div className="flex gap-3 mt-6">
          <button className="px-4 py-2 bg-emerald-500 hover:bg-emerald-400 text-gray-950 rounded-lg text-sm font-medium transition">Cambiar plan</button>
          <button className="px-4 py-2 border border-red-500/30 text-red-400 hover:bg-red-500/10 rounded-lg text-sm transition">Cancelar suscripción</button>
        </div>
      </div>

      <div className="p-6 border border-gray-800 rounded-xl">
        <h2 className="font-semibold mb-2">¿Necesitás más instancias?</h2>
        <p className="text-sm text-gray-400 mb-4">Cada licencia Pro incluye 3 instancias. Si necesitás más, upgradé a Enterprise.</p>
        <a href="/checkout?tier=enterprise" className="text-emerald-400 hover:text-emerald-300 text-sm font-medium">Ver planes Enterprise →</a>
      </div>
    </div>
  );
}
