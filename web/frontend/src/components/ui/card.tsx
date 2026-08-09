"use client";

import { motion } from "framer-motion";

interface CardProps {
  children: React.ReactNode;
  className?: string;
  hover?: boolean;
  glass?: boolean;
  bordered?: boolean;
}

export function Card({ children, className = "", hover = false, glass = false, bordered = true }: CardProps) {
  return (
    <motion.div
      className={`rounded-2xl p-6 transition-colors duration-200 ${
        glass
          ? "bg-white/[0.03] backdrop-blur-xl border border-white/[0.06]"
          : bordered
          ? "border border-gray-800 bg-gray-950/50"
          : "bg-gray-950/50"
      } ${hover ? "hover:border-emerald-500/30 hover:bg-emerald-500/[0.02]" : ""} ${className}`}
      whileHover={hover ? { y: -2 } : undefined}
      transition={{ duration: 0.2 }}
    >
      {children}
    </motion.div>
  );
}
