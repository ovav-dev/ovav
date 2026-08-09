"use client";

const INVOICES = [
  { id: "inv_1Q2W3E4R", date: "15 Jun 2026", amount: "$180.00", status: "paid", pdf: "#" },
  { id: "inv_2W3E4R5T", date: "15 May 2026", amount: "$180.00", status: "paid", pdf: "#" },
  { id: "inv_3E4R5T6Y", date: "15 Abr 2026", amount: "$180.00", status: "paid", pdf: "#" },
  { id: "inv_4R5T6Y7U", date: "15 Mar 2026", amount: "$180.00", status: "paid", pdf: "#" },
  { id: "inv_5T6Y7U8I", date: "15 Feb 2026", amount: "$180.00", status: "paid", pdf: "#" },
  { id: "inv_6Y7U8I9O", date: "15 Ene 2026", amount: "$180.00", status: "paid", pdf: "#" },
];

export default function InvoicesPage() {
  return (
    <div className="max-w-2xl mx-auto px-6 py-12">
      <h1 className="text-2xl font-bold mb-8">Facturación</h1>

      <div className="border border-gray-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800 bg-gray-900/50 text-left text-gray-500">
              <th className="py-3 px-4 font-medium">Factura</th>
              <th className="py-3 px-4 font-medium">Fecha</th>
              <th className="py-3 px-4 font-medium">Importe</th>
              <th className="py-3 px-4 font-medium">Estado</th>
              <th className="py-3 px-4 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            {INVOICES.map((inv) => (
              <tr key={inv.id} className="border-b border-gray-800/50 hover:bg-gray-900/30">
                <td className="py-3 px-4 font-mono text-xs text-gray-300">{inv.id}</td>
                <td className="py-3 px-4 text-gray-400">{inv.date}</td>
                <td className="py-3 px-4 text-gray-200">{inv.amount}</td>
                <td className="py-3 px-4">
                  <span className="text-xs px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-400">{inv.status}</span>
                </td>
                <td className="py-3 px-4">
                  <a href={inv.pdf} className="text-xs text-emerald-400 hover:text-emerald-300">Descargar PDF</a>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className="text-xs text-gray-600 mt-4">Facturas emitidas por OVAV Technologies, Buenos Aires, Argentina. CUIT: 30-XXXXXXXX-X.</p>
    </div>
  );
}
