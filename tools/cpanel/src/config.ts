// OVAV cPanel — Runtime config
// Use VITE_API_BASE env var for dev/prod switching.
// Production: https://d678beea.ovav.dev
// Dev local:  http://localhost:5858 (or Vite proxy)
export const API_BASE = import.meta.env.VITE_API_BASE || 'https://d678beea.ovav.dev';
