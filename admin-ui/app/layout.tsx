import type { Metadata } from "next";
import "./globals.css";
import Link from "next/link";

export const metadata: Metadata = { title: "LLM Relay Admin" };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="bg-gray-950 text-gray-100 min-h-screen">
        <nav className="border-b border-gray-800 px-6 py-4 flex gap-6 items-center">
          <span className="font-bold text-white text-lg">LLM Relay</span>
          <Link href="/" className="text-gray-400 hover:text-white text-sm">Dashboard</Link>
          <Link href="/keys" className="text-gray-400 hover:text-white text-sm">API Keys</Link>
          <Link href="/projects" className="text-gray-400 hover:text-white text-sm">Projects</Link>
        </nav>
        <main className="max-w-5xl mx-auto px-6 py-8">{children}</main>
      </body>
    </html>
  );
}
