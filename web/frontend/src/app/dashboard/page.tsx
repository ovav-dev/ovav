'use client';

import Link from 'next/link';
import { useAuth } from '@/lib/auth';
import api from '@/lib/api';

export default function DashboardPage() {
  const { user } = useAuth();
  const token = api.getToken();

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Welcome back, {user?.name || 'User'} 👋</h1>
        <p className="text-gray-400 mt-1">Here&apos;s an overview of your OVAV account.</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="Subscription Tier"
          value="Pro"
          icon="💎"
          color="violet"
        />
        <StatCard
          title="API Keys"
          value="0"
          icon="🔑"
          color="blue"
        />
        <StatCard
          title="Instances Active"
          value="1 / 3"
          icon="🖥️"
          color="green"
        />
        <StatCard
          title="Days Left"
          value="23"
          icon="📅"
          color="amber"
        />
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Quick Actions</h2>
          <div className="space-y-3">
            <QuickAction
              href="/dashboard/api-keys"
              title="Create API Key"
              description="Generate a new API key for your applications"
              icon="🔑"
            />
            <QuickAction
              href="/dashboard/download"
              title="Download CLI"
              description="Get the latest OVAV CLI for your platform"
              icon="⬇️"
            />
            <QuickAction
              href="/dashboard/subscription"
              title="Manage Subscription"
              description="View or change your current plan"
              icon="💎"
            />
          </div>
        </div>

        {/* Token Status */}
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Session Status</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Token</span>
              <span className={`px-2 py-1 rounded text-xs font-medium ${
                token ? 'bg-green-900 text-green-300' : 'bg-red-900 text-red-300'
              }`}>
                {token ? 'Active' : 'None'}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Email</span>
              <span className="text-sm">{user?.email || 'Not authenticated'}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Email Verified</span>
              <span className="px-2 py-1 rounded text-xs font-medium bg-green-900 text-green-300">
                Verified
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Debug Info (Development only) */}
      {process.env.NODE_ENV === 'development' && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Debug Info</h2>
          <pre className="text-xs text-gray-400 overflow-auto">
            {JSON.stringify({ user, hasToken: !!token }, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

function StatCard({
  title,
  value,
  icon,
  color,
}: {
  title: string;
  value: string;
  icon: string;
  color: string;
}) {
  const colorClasses: Record<string, string> = {
    violet: 'bg-violet-900/50 text-violet-400 border-violet-800',
    blue: 'bg-blue-900/50 text-blue-400 border-blue-800',
    green: 'bg-green-900/50 text-green-400 border-green-800',
    amber: 'bg-amber-900/50 text-amber-400 border-amber-800',
  };

  return (
    <div className={`bg-gray-900 border ${colorClasses[color]} rounded-xl p-6`}>
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-400">{title}</p>
          <p className="text-2xl font-bold mt-1">{value}</p>
        </div>
        <span className="text-3xl">{icon}</span>
      </div>
    </div>
  );
}

function QuickAction({
  href,
  title,
  description,
  icon,
}: {
  href: string;
  title: string;
  description: string;
  icon: string;
}) {
  return (
    <a
      href={href}
      className="flex items-center gap-4 p-4 bg-gray-800 rounded-lg hover:bg-gray-750 transition-colors"
    >
      <span className="text-2xl">{icon}</span>
      <div>
        <p className="font-medium">{title}</p>
        <p className="text-sm text-gray-400">{description}</p>
      </div>
    </a>
  );
}
