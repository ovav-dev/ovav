import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        ovav: {
          bg: "#0d1117",
          surface: "#161b22",
          border: "#30363d",
          accent: "#fe8019",       // orange — Dante's color, primary CTA
          accent2: "#8ec07c",      // green — Thavren/Platform
          accent3: "#83a598",      // teal — Eidren/Evidence
          accent4: "#d3869b",      // rose — Valeria/Education
          accent5: "#b8bb26",      // lime — Renata/Health
          accent6: "#fabd2f",      // yellow — Sofía/Commercial
          accent7: "#928374",      // gray — neutral
          text: "#e6edf3",
          muted: "#8b949e",
        },
      },
      fontFamily: {
        mono: ["JetBrains Mono", "Fira Code", "monospace"],
        sans: ["Inter", "system-ui", "sans-serif"],
      },
    },
  },
  plugins: [],
};
export default config;
