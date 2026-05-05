"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export default function NewContract() {
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!file) {
      setErr("Choose a PDF or DOCX file.");
      return;
    }
    setSubmitting(true);
    setErr(null);
    const fd = new FormData();
    fd.append("title", title);
    fd.append("file", file);

    const apiBase =
      process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8000";
    const res = await fetch(`${apiBase}/contracts`, {
      method: "POST",
      credentials: "include",
      body: fd,
    });
    setSubmitting(false);
    if (!res.ok) {
      const text = await res.text();
      setErr(text || `Upload failed (${res.status})`);
      return;
    }
    const json = (await res.json()) as { id: string };
    router.push(`/contracts/${json.id}`);
  }

  return (
    <main className="flex min-h-screen flex-col gap-6 p-8">
      <header className="border-b border-neutral-200 pb-3">
        <h1 className="text-xl font-semibold">New contract</h1>
        <p className="text-xs text-neutral-500">PDF or DOCX, up to 25 MB.</p>
      </header>
      <form className="flex max-w-md flex-col gap-3" onSubmit={onSubmit}>
        <label className="text-sm font-medium">Title</label>
        <input
          required
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="rounded-md border border-neutral-300 px-3 py-2"
          placeholder="Acme x Beta NDA"
        />
        <label className="text-sm font-medium">Document</label>
        <input
          required
          type="file"
          accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          className="rounded-md border border-neutral-300 px-3 py-2"
        />
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-neutral-900 px-3 py-2 text-white disabled:opacity-50"
        >
          {submitting ? "Uploading..." : "Upload + extract"}
        </button>
        {err && <p className="text-sm text-red-600">{err}</p>}
      </form>
    </main>
  );
}
