import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Pactline",
  description: "Contract lifecycle platform built on durable workflows.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="antialiased">{children}</body>
    </html>
  );
}
