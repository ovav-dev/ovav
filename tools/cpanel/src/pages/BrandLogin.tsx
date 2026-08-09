import { useEffect } from 'react';

// Cloudflare Access login URL for this app's domain
const CF_ACCESS_LOGIN_URL = 'https://d678beea.cloudflareaccess.com/cdn-cgi/access/login/d678beea.ovav.dev';

export default function BrandLogin() {
  // Auto-redirect after brief branded intro (3 s)
  useEffect(() => {
    let count = 3;
    const el = document.getElementById('countdown');
    if (el) el.textContent = String(count);

    const interval = setInterval(() => {
      count -= 1;
      if (el) el.textContent = String(count);
      if (count <= 0) clearInterval(interval);
    }, 1000);

    const t = setTimeout(() => {
      window.location.href = CF_ACCESS_LOGIN_URL;
    }, 3000);
    return () => { clearTimeout(t); clearInterval(interval); };
  }, []);

  return (
    <div style={{
      minHeight: '100vh',
      background: 'linear-gradient(135deg, #0f0f1a 0%, #1a1035 50%, #0f0f1a 100%)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      fontFamily: "'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
      color: '#e2e8f0',
    }}>
      {/* Background grid */}
      <div style={{
        position: 'fixed',
        inset: 0,
        backgroundImage: `
          linear-gradient(rgba(124,58,237,0.07) 1px, transparent 1px),
          linear-gradient(90deg, rgba(124,58,237,0.07) 1px, transparent 1px)
        `,
        backgroundSize: '48px 48px',
        pointerEvents: 'none',
      }} />

      {/* Glow blobs */}
      <div style={{
        position: 'fixed',
        top: '15%',
        left: '20%',
        width: 400,
        height: 400,
        background: 'radial-gradient(circle, rgba(124,58,237,0.18) 0%, transparent 70%)',
        pointerEvents: 'none',
      }} />
      <div style={{
        position: 'fixed',
        bottom: '10%',
        right: '15%',
        width: 300,
        height: 300,
        background: 'radial-gradient(circle, rgba(139,92,246,0.12) 0%, transparent 70%)',
        pointerEvents: 'none',
      }} />

      <div style={{
        position: 'relative',
        textAlign: 'center',
        maxWidth: 480,
        padding: '0 24px',
      }}>
        {/* Logo mark */}
        <div style={{ marginBottom: 32 }}>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 128 128"
            width="80"
            height="80"
            style={{ display: 'block', margin: '0 auto' }}
          >
            <defs>
              <linearGradient id="blg" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stopColor="#7c3aed" />
                <stop offset="100%" stopColor="#8b5cf6" />
              </linearGradient>
              <filter id="glow">
                <feGaussianBlur stdDeviation="4" result="coloredBlur" />
                <feMerge>
                  <feMergeNode in="coloredBlur" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
            </defs>
            <rect width="128" height="128" rx="24" fill="url(#blg)" filter="url(#glow)" />
            <text
              x="64" y="82"
              fontFamily="system-ui, sans-serif"
              fontSize="56"
              fontWeight="700"
              fill="white"
              textAnchor="middle"
              letterSpacing="-2"
            >
              OV
            </text>
            <circle cx="96" cy="40" r="8" fill="#f59e0b" />
          </svg>
        </div>

        {/* Brand wordmark */}
        <h1 style={{
          fontSize: 32,
          fontWeight: 800,
          letterSpacing: '-0.5px',
          background: 'linear-gradient(135deg, #7c3aed, #8b5cf6, #a78bfa)',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent',
          marginBottom: 8,
          lineHeight: 1.2,
        }}>
          OVAV Systems
        </h1>
        <p style={{
          fontSize: 14,
          color: '#64748b',
          letterSpacing: '0.5px',
          textTransform: 'uppercase',
          marginBottom: 40,
        }}>
          cPanel Access
        </p>

        {/* Divider */}
        <div style={{
          height: 1,
          background: 'linear-gradient(90deg, transparent, rgba(124,58,237,0.4), transparent)',
          marginBottom: 40,
        }} />

        {/* Status indicator */}
        <div style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 8,
          padding: '8px 16px',
          background: 'rgba(124,58,237,0.1)',
          border: '1px solid rgba(124,58,237,0.3)',
          borderRadius: 999,
          fontSize: 13,
          color: '#a78bfa',
          marginBottom: 32,
        }}>
          <span style={{
            display: 'inline-block',
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: '#7c3aed',
            boxShadow: '0 0 8px #7c3aed',
            animation: 'brandPulse 2s ease-in-out infinite',
          }} />
          Redirecting to Cloudflare Access
        </div>

        {/* Primary CTA */}
        <a
          href={CF_ACCESS_LOGIN_URL}
          style={{
            display: 'block',
            padding: '14px 32px',
            background: 'linear-gradient(135deg, #7c3aed, #6d28d9)',
            border: '1px solid rgba(139,92,246,0.5)',
            borderRadius: 10,
            color: '#fff',
            fontSize: 15,
            fontWeight: 700,
            fontFamily: 'inherit',
            textDecoration: 'none',
            letterSpacing: '0.3px',
            boxShadow: '0 4px 24px rgba(124,58,237,0.35)',
            transition: 'all 0.2s ease',
            cursor: 'pointer',
            marginBottom: 16,
          }}
          onMouseEnter={e => {
            (e.currentTarget as HTMLAnchorElement).style.background = 'linear-gradient(135deg, #6d28d9, #5b21b6)';
            (e.currentTarget as HTMLAnchorElement).style.boxShadow = '0 6px 32px rgba(124,58,237,0.5)';
            (e.currentTarget as HTMLAnchorElement).style.transform = 'translateY(-1px)';
          }}
          onMouseLeave={e => {
            (e.currentTarget as HTMLAnchorElement).style.background = 'linear-gradient(135deg, #7c3aed, #6d28d9)';
            (e.currentTarget as HTMLAnchorElement).style.boxShadow = '0 4px 24px rgba(124,58,237,0.35)';
            (e.currentTarget as HTMLAnchorElement).style.transform = 'translateY(0)';
          }}
        >
          Continue with Cloudflare Access
        </a>

        {/* Domain note */}
        <p style={{ fontSize: 12, color: '#475569', marginBottom: 0 }}>
          Secured by Cloudflare Zero Trust · d678beea.ovav.dev
        </p>

        {/* Auto-redirect countdown */}
        <p style={{ fontSize: 11, color: '#334155', marginTop: 24 }}>
          Auto-redirect in{' '}
          <span
            id="countdown"
            style={{ color: '#7c3aed', fontWeight: 700 }}
          >
            3
          </span>
          {' '}s
        </p>
      </div>

      <style>{`
        @keyframes brandPulse {
          0%, 100% { opacity: 1; box-shadow: 0 0 8px #7c3aed; }
          50% { opacity: 0.6; box-shadow: 0 0 3px #7c3aed; }
        }
      `}</style>
    </div>
  );
}
