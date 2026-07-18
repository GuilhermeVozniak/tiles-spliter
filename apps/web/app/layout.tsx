import type { Metadata } from "next";
import { Inter } from "next/font/google";
import Link from "next/link";
import "./globals.css";

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" });

const RELEASES_URL =
  "https://github.com/GuilhermeVozniak/tiles-spliter/releases/latest";
const REPO_URL = "https://github.com/GuilhermeVozniak/tiles-spliter";

export const metadata: Metadata = {
  title: "Tiles Spliter — window manager for macOS",
  description:
    "Snap, tile and arrange your Mac windows — by drag, hotkey or menu bar. A fast, native window manager for macOS 13+.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={inter.variable}>
      <body className="min-h-screen bg-zinc-950 font-sans text-zinc-100 antialiased">
        <header className="border-b border-zinc-900">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
            <Link
              href="/"
              className="text-sm font-semibold tracking-tight text-zinc-100"
            >
              Tiles Spliter
            </Link>
            <nav className="flex items-center gap-6 text-sm text-zinc-400">
              <Link
                href="/changelog"
                className="transition-colors hover:text-zinc-100"
              >
                Changelog
              </Link>
              <Link
                href="/privacy"
                className="transition-colors hover:text-zinc-100"
              >
                Privacy
              </Link>
              <a
                href={REPO_URL}
                target="_blank"
                rel="noreferrer"
                className="transition-colors hover:text-zinc-100"
              >
                GitHub
              </a>
            </nav>
          </div>
        </header>
        <main>{children}</main>
        <footer className="border-t border-zinc-900">
          <div className="mx-auto flex max-w-5xl flex-col items-center justify-between gap-2 px-6 py-8 text-xs text-zinc-500 sm:flex-row">
            <span>
              &copy; {new Date().getFullYear()} Tiles Spliter. Made for macOS.
            </span>
            <a
              href={RELEASES_URL}
              target="_blank"
              rel="noreferrer"
              className="hover:text-zinc-300"
            >
              Latest release
            </a>
          </div>
        </footer>
      </body>
    </html>
  );
}
