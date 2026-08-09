import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "OVAV — AI Workstation Governor | Governed AI, Not Just Suggested AI",
  description:
    "OVAV governs how you use AI across your entire development lifecycle. Eight professional profiles, multi-model without vendor lock-in, 100% local-first. Plan → Build → Test → Deploy.",
  keywords: [
    "OVAV", "AI workstation governor", "AI governance", "multi-model AI",
    "local-first AI", "developer tools", "software engineering", "AI orchestration",
    "Go runtime", "TypeScript", "open-source AI", "SDLC governance",
  ],
  authors: [{ name: "OVAV", url: "https://ovav.dev" }],
  openGraph: {
    title: "OVAV — AI Workstation Governor",
    description:
      "Govern your entire development lifecycle with AI. Eight professional profiles, multi-model, local-first. The operating system for AI-augmented software engineering.",
    url: "https://ovav.dev",
    siteName: "OVAV",
    locale: "en_US",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "OVAV — AI Workstation Governor",
    description:
      "Govern your entire development lifecycle with AI. Eight professional profiles, multi-model, local-first.",
  },
  robots: {
    index: true,
    follow: true,
  },
  alternates: {
    canonical: "https://ovav.dev",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-ovav-bg text-ovav-text antialiased">
        {children}
      </body>
    </html>
  );
}
