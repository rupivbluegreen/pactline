"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { client } from "@/api/client";

export default function SignIn() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 p-8">
      <h1 className="text-3xl font-semibold">Pactline</h1>
      <form
        className="flex w-80 flex-col gap-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setSubmitting(true);
          setErr(null);
          const { error } = await client.POST("/auth/magic-link/request", {
            body: { email },
          });
          setSubmitting(false);
          if (error) {
            setErr(String(error));
          } else {
            router.push("/signin/sent");
          }
        }}
      >
        <label className="text-sm font-medium">Work email</label>
        <input
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="rounded-md border border-neutral-300 px-3 py-2"
          placeholder="you@company.com"
        />
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-neutral-900 px-3 py-2 text-white disabled:opacity-50"
        >
          {submitting ? "Sending..." : "Send sign-in link"}
        </button>
        {err && <p className="text-sm text-red-600">{err}</p>}
      </form>
    </main>
  );
}
