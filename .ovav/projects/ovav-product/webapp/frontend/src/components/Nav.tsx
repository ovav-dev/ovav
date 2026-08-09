"use client";

import { useState } from "react";

const NAV_LINKS = [
  { href: "#pricing", label: "Pricing" },
  { href: "#profiles", label: "Profiles" },
  { href: "#moat", label: "Why OVAV" },
  { href: "#cta", label: "Get Access" },
];

export function Nav() {
  const [open, setOpen] = useState(false);

  return (
    <nav className="fixed top-0 left-0 right-0 z-50 nav-blur bg-ovav-bg/80 border-b border-ovav-border">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <a href="/" className="flex items-center gap-2 font-bold text-xl">
            <span className="text-ovav-accent text-2xl">◆</span>
            <span className="text-white">OVAV</span>
          </a>

          {/* Desktop nav */}
          <div className="hidden md:flex items-center gap-8">
            {NAV_LINKS.map((link) => (
              <a
                key={link.href}
                href={link.href}
                className="text-sm text-ovav-muted hover:text-ovav-text transition-colors"
              >
                {link.label}
              </a>
            ))}
            <a
              href="https://cpanel.ovav.dev"
              className="btn-primary text-sm !px-4 !py-2"
            >
              cPanel →
            </a>
          </div>

          {/* Mobile menu button */}
          <button
            className="md:hidden p-2 text-ovav-muted"
            onClick={() => setOpen(!open)}
            aria-label="Toggle menu"
          >
            {open ? (
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
            ) : (
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M3 12h18M3 6h18M3 18h18"/></svg>
            )}
          </button>
        </div>

        {/* Mobile menu */}
        {open && (
          <div className="md:hidden pb-4 border-t border-ovav-border mt-2 pt-4 space-y-3">
            {NAV_LINKS.map((link) => (
              <a
                key={link.href}
                href={link.href}
                className="block text-sm text-ovav-muted hover:text-ovav-text py-2"
                onClick={() => setOpen(false)}
              >
                {link.label}
              </a>
            ))}
            <a
              href="https://cpanel.ovav.dev"
              className="btn-primary text-sm w-full !justify-center mt-4"
            >
              cPanel Access →
            </a>
          </div>
        )}
      </div>
    </nav>
  );
}
