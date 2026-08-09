'use client';

import { useState, useEffect } from 'react';
import { useAuth } from '@/lib/auth';
import api, { Subscription } from '@/lib/api';

const tiers = [
  { id: 'core', name: 'Core', price: 'Free', features: ['3 instances', 'Basic support', 'Community access'] },
  { id: 'pro', name: 'Pro', price: '$29/mo', features: ['10 instances', 'Priority support', 'API access', 'Advanced features'] },
  { id: 'enterprise', name: 'Enterprise', price: '$199/mo', features: ['Unlimited instances', '24/7 support', 'SSO', 'SLA guarantee'] },
];

export default function SubscriptionPage() {
  const { user } = useAuth();
  const [subscription, setSubscription] = useState<Subscription | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isUpdating, setIsUpdating] = useState(false);

  useEffect(() => {
    if (api.isAuthenticated()) {
      loadSubscription();
    } else {
      setIsLoading(false);
    }
  }, []);

  async function loadSubscription() {
    try {
      const data = await api.getSubscription();
      setSubscription(data);
    } catch (error) {
      console.error('Failed to load subscription:', error);
    } finally {
      setIsLoading(false);
    }
  }

  async function handleChangeTier(tierId: string) {
    setIsUpdating(true);
    try {
      await api.updateSubscription({ tier: tierId });
      await loadSubscription();
    } catch (error) {
      console.error('Failed to change tier:', error);
    } finally {
      setIsUpdating(false);
    }
  }

  async function handleCancel() {
    if (!confirm('Are you sure you want to cancel your subscription?')) return;

    setIsUpdating(true);
    try {
      await api.updateSubscription({ cancel: true });
      await loadSubscription();
    } catch (error) {
      console.error('Failed to cancel:', error);
    } finally {
      setIsUpdating(false);
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin w-8 h-8 border-4 border-violet-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  const currentTier = subscription?.tier || 'core';

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Subscription</h1>
        <p className="text-gray-400 mt-1">Manage your OVAV subscription plan.</p>
      </div>

      {/* Current Plan */}
      <div className="bg-gradient-to-r from-violet-900/50 to-fuchsia-900/50 border border-violet-800 rounded-xl p-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-violet-300">Current Plan</p>
            <p className="text-3xl font-bold mt-1">
              {tiers.find(t => t.id === currentTier)?.name || 'Core'}
            </p>
            <p className="text-gray-400 mt-1">
              {subscription?.status === 'active' ? 'Active' : subscription?.status || 'Free tier'}
            </p>
          </div>
          <div className="text-right">
            <p className="text-2xl font-bold">
              {tiers.find(t => t.id === currentTier)?.price || 'Free'}
            </p>
            {subscription?.current_period_end && (
              <p className="text-sm text-gray-400 mt-1">
                Renews {new Date(subscription.current_period_end).toLocaleDateString()}
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Plan Comparison */}
      <div>
        <h2 className="text-lg font-semibold mb-4">Available Plans</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {tiers.map((tier) => (
            <div
              key={tier.id}
              className={`bg-gray-900 border rounded-xl p-6 ${
                tier.id === currentTier
                  ? 'border-violet-500 ring-2 ring-violet-500/20'
                  : 'border-gray-800'
              }`}
            >
              <h3 className="text-xl font-bold">{tier.name}</h3>
              <p className="text-2xl font-bold mt-2">{tier.price}</p>
              
              <ul className="mt-4 space-y-2">
                {tier.features.map((feature, i) => (
                  <li key={i} className="text-sm text-gray-400 flex items-center gap-2">
                    <span className="text-green-400">✓</span> {feature}
                  </li>
                ))}
              </ul>

              <button
                onClick={() => handleChangeTier(tier.id)}
                disabled={tier.id === currentTier || isUpdating}
                className={`w-full mt-6 px-4 py-2 rounded-lg font-medium transition-colors ${
                  tier.id === currentTier
                    ? 'bg-gray-700 text-gray-400 cursor-not-allowed'
                    : 'bg-violet-600 hover:bg-violet-700'
                }`}
              >
                {tier.id === currentTier ? 'Current Plan' : 'Switch to ' + tier.name}
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Cancel */}
      {currentTier !== 'core' && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Cancel Subscription</p>
              <p className="text-sm text-gray-400 mt-1">Your access will continue until the end of the billing period.</p>
            </div>
            <button
              onClick={handleCancel}
              disabled={isUpdating}
              className="px-4 py-2 text-red-400 border border-red-800 hover:bg-red-900/30 rounded-lg font-medium transition-colors"
            >
              Cancel Plan
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
