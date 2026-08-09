'use client';

import { useState, useEffect } from 'react';
import api, { DownloadUrls } from '@/lib/api';

const platformIcons: Record<string, string> = {
  macos: '🍎',
  linux: '🐧',
  windows: '🪟',
};

const platformNames: Record<string, string> = {
  macos: 'macOS',
  linux: 'Linux',
  windows: 'Windows',
};

export default function DownloadPage() {
  const [downloadData, setDownloadData] = useState<DownloadUrls | null>(null);
  const [detected, setDetected] = useState<{ platform: string; arch: string } | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [urls, platform] = await Promise.all([
          api.getDownloadUrls(),
          api.detectPlatform(),
        ]);
        setDownloadData(urls);
        setDetected({ platform: platform.platform, arch: platform.arch });
      } catch (error) {
        console.error('Failed to load download info:', error);
      } finally {
        setIsLoading(false);
      }
    }
    load();
  }, []);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin w-8 h-8 border-4 border-violet-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  const version = downloadData?.version || '3.0.0';
  
  // Group by platform
  const byPlatform = downloadData?.platforms.reduce((acc, p) => {
    if (!acc[p.platform]) acc[p.platform] = [];
    acc[p.platform].push(p);
    return acc;
  }, {} as Record<string, typeof downloadData.platforms>) || {};

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Download OVAV CLI</h1>
        <p className="text-gray-400 mt-1">Get the latest version for your platform.</p>
      </div>

      {/* Version Badge */}
      <div className="inline-flex items-center gap-2 px-4 py-2 bg-gray-900 border border-gray-800 rounded-full">
        <span className="text-green-400">●</span>
        <span className="text-sm">Version {version}</span>
        <span className="text-gray-500">·</span>
        <span className="text-sm text-gray-400">Released today</span>
      </div>

      {/* Recommended Download */}
      {detected && (
        <div className="bg-gradient-to-r from-violet-900/50 to-fuchsia-900/50 border border-violet-800 rounded-xl p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <span className="text-4xl">{platformIcons[detected.platform] || '💻'}</span>
              <div>
                <p className="text-sm text-violet-300">Recommended Download</p>
                <p className="text-xl font-bold">
                  {platformNames[detected.platform]} ({detected.arch})
                </p>
              </div>
            </div>
            <a
              href={`/download/ovav-${detected.platform}-${detected.arch}`}
              className="px-6 py-3 bg-white text-gray-900 font-bold rounded-lg hover:bg-gray-100 transition-colors"
            >
              Download
            </a>
          </div>
        </div>
      )}

      {/* All Downloads */}
      <div>
        <h2 className="text-lg font-semibold mb-4">All Platforms</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {Object.entries(byPlatform).map(([platform, downloads]) => (
            <div key={platform} className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
              <div className="px-6 py-4 bg-gray-800 flex items-center gap-3">
                <span className="text-2xl">{platformIcons[platform] || '💻'}</span>
                <span className="font-semibold">{platformNames[platform] || platform}</span>
              </div>
              <div className="divide-y divide-gray-800">
                {downloads.map((download) => (
                  <div key={download.arch} className="px-6 py-3 flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium">{download.arch}</p>
                      <p className="text-xs text-gray-500">{download.size_mb} MB</p>
                    </div>
                    <a
                      href={download.url}
                      className="px-3 py-1.5 text-sm bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
                    >
                      Download
                    </a>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Installation Instructions */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Installation</h2>
        
        <div className="space-y-4">
          <div>
            <p className="text-sm font-medium mb-2">macOS / Linux:</p>
            <code className="block p-4 bg-gray-800 rounded-lg text-sm overflow-x-auto">
              curl -fsSL https://get.ovav.dev | sh
            </code>
          </div>
          
          <div>
            <p className="text-sm font-medium mb-2">Windows (PowerShell):</p>
            <code className="block p-4 bg-gray-800 rounded-lg text-sm overflow-x-auto">
              iwr https://get.ovav.dev/install.ps1 -useb | iex
            </code>
          </div>
        </div>
      </div>

      {/* Verify Installation */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Verify Installation</h2>
        <code className="block p-4 bg-gray-800 rounded-lg text-sm">
          ovav --version
        </code>
        <p className="text-sm text-gray-400 mt-2">
          Should output: <span className="text-green-400">OVAV v{version}</span>
        </p>
      </div>
    </div>
  );
}
