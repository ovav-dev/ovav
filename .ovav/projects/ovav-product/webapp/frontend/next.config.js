/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  images: {
    unoptimized: true,
  },
  // Trailing slash for Cloudflare Pages compatibility
  trailingSlash: true,
};

module.exports = nextConfig;
