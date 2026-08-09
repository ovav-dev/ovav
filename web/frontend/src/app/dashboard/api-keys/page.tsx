'use client';

import { useState, useEffect } from 'react';
import { useAuth } from '@/lib/auth';
import api, { ApiKey, ApiKeyCreated } from '@/lib/api';

export default function ApiKeysPage() {
  const { user } = useAuth();
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [newKey, setNewKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (api.isAuthenticated()) {
      loadKeys();
    } else {
      setIsLoading(false);
    }
  }, []);

  async function loadKeys() {
    try {
      setIsLoading(true);
      const data = await api.listApiKeys();
      setKeys(data);
      setError(null);
    } catch (err) {
      setError('Failed to load API keys');
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  }

  async function handleCreateKey() {
    if (!newKeyName.trim()) return;

    try {
      const created: ApiKeyCreated = await api.createApiKey(newKeyName);
      setNewKey(created.key); // Show full key only once!
      setNewKeyName('');
      loadKeys();
      setShowCreateModal(false);
    } catch (err) {
      setError('Failed to create API key');
      console.error(err);
    }
  }

  async function handleDeleteKey(keyId: string) {
    if (!confirm('Are you sure you want to delete this API key?')) return;

    try {
      await api.deleteApiKey(keyId);
      loadKeys();
    } catch (err) {
      setError('Failed to delete API key');
      console.error(err);
    }
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text);
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
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">API Keys</h1>
          <p className="text-gray-400 mt-1">Manage your API keys for programmatic access.</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="px-4 py-2 bg-violet-600 hover:bg-violet-700 rounded-lg font-medium transition-colors"
        >
          Create New Key
        </button>
      </div>

      {/* Error */}
      {error && (
        <div className="p-4 bg-red-900/50 border border-red-800 rounded-lg text-red-300">
          {error}
        </div>
      )}

      {/* New Key Modal */}
      {newKey && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-amber-400 flex items-center gap-2">
              ⚠️ Save Your API Key
            </h3>
            <p className="text-gray-400 mt-2 text-sm">
              Copy this key now. You won&apos;t be able to see it again!
            </p>
            <div className="mt-4 p-3 bg-gray-800 rounded-lg font-mono text-sm break-all">
              {newKey}
            </div>
            <div className="flex gap-3 mt-4">
              <button
                onClick={() => copyToClipboard(newKey)}
                className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg font-medium transition-colors"
              >
                Copy
              </button>
              <button
                onClick={() => setNewKey(null)}
                className="flex-1 px-4 py-2 bg-violet-600 hover:bg-violet-700 rounded-lg font-medium transition-colors"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold">Create New API Key</h3>
            <div className="mt-4">
              <label className="block text-sm text-gray-400 mb-2">Key Name</label>
              <input
                type="text"
                value={newKeyName}
                onChange={(e) => setNewKeyName(e.target.value)}
                placeholder="e.g., Production API"
                className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg focus:outline-none focus:border-violet-500"
              />
            </div>
            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setShowCreateModal(false)}
                className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg font-medium transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateKey}
                disabled={!newKeyName.trim()}
                className="flex-1 px-4 py-2 bg-violet-600 hover:bg-violet-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg font-medium transition-colors"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Keys List */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        {keys.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            <p>No API keys yet. Create one to get started.</p>
          </div>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-800">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Key Prefix</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Created</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Expires</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-400 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              {keys.map((key) => (
                <tr key={key.id} className="hover:bg-gray-800/50">
                  <td className="px-6 py-4 font-medium">{key.name}</td>
                  <td className="px-6 py-4 font-mono text-sm text-gray-400">
                    ovav_****_{key.key_prefix}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-400">
                    {new Date(key.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-400">
                    {key.expires_at
                      ? new Date(key.expires_at).toLocaleDateString()
                      : 'Never'}
                  </td>
                  <td className="px-6 py-4 text-right">
                    <button
                      onClick={() => handleDeleteKey(key.id)}
                      className="text-red-400 hover:text-red-300 transition-colors"
                    >
                      Delete
                    </button>
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
