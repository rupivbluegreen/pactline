"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export function StatusPoller({ status }: { status: string }) {
  const router = useRouter();
  useEffect(() => {
    if (status !== "intake" && status !== "parsing") return;
    const t = window.setInterval(() => router.refresh(), 2000);
    return () => window.clearInterval(t);
  }, [status, router]);
  return null;
}
