'use client';

import { useState, useEffect } from 'react';
import { useAuth } from '@/lib/auth';
import api, { Invoice } from '@/lib/api';

export default function BillingPage() {
  const { user } = useAuth();
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isOpeningPortal, setIsOpeningPortal] = useState(false);

  useEffect(() => {
    if (api.isAuthenticated()) {
      loadInvoices();
    } else {
      setIsLoading(false);
    }
  }, []);

  async function loadInvoices() {
    try {
      const data = await api.listInvoices();
      setInvoices(data);
    } catch (error) {
      console.error('Failed to load invoices:', error);
    } finally {
      setIsLoading(false);
    }
  }

  async function handleOpenPortal() {
    setIsOpeningPortal(true);
    try {
      const { url } = await api.createPortalSession();
      window.open(url, '_blank');
    } catch (error) {
      console.error('Failed to open portal:', error);
    } finally {
      setIsOpeningPortal(false);
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin w-8 h-8 border-4 border-violet-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Billing</h1>
          <p className="text-gray-400 mt-1">Manage your invoices and payment methods.</p>
        </div>
        <button
          onClick={handleOpenPortal}
          disabled={isOpeningPortal}
          className="px-4 py-2 bg-violet-600 hover:bg-violet-700 disabled:opacity-50 rounded-lg font-medium transition-colors"
        >
          {isOpeningPortal ? 'Opening...' : 'Manage Payment Methods'}
        </button>
      </div>

      {/* Invoices */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-800">
          <h2 className="font-semibold">Invoice History</h2>
        </div>
        
        {invoices.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            <p>No invoices yet.</p>
            <p className="text-sm mt-1">Invoices will appear here once you make a payment.</p>
          </div>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-800">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Date</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Amount</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Status</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-400 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              {invoices.map((invoice) => (
                <tr key={invoice.id} className="hover:bg-gray-800/50">
                  <td className="px-6 py-4 text-sm">
                    {new Date(invoice.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 font-medium">
                    ${invoice.amount.toFixed(2)} {invoice.currency.toUpperCase()}
                  </td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${
                      invoice.status === 'paid'
                        ? 'bg-green-900/50 text-green-400'
                        : invoice.status === 'pending'
                        ? 'bg-amber-900/50 text-amber-400'
                        : 'bg-red-900/50 text-red-400'
                    }`}>
                      {invoice.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right">
                    {invoice.invoice_url && (
                      <a
                        href={invoice.invoice_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-violet-400 hover:text-violet-300 transition-colors"
                      >
                        View
                      </a>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
