"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { client } from "@/api/client";

export default function OnboardingOrganization() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 p-8">
      <h1 className="text-2xl font-semibold">Create your organization</h1>
      <form
        className="flex w-80 flex-col gap-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setSubmitting(true);
          setErr(null);
          const { error } = await client.POST("/organizations", {
            body: { name },
          });
          setSubmitting(false);
          if (error) {
            setErr(String(error));
            return;
          }
          router.push("/contracts");
        }}
      >
        <input
          type="text"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="rounded-md border border-neutral-300 px-3 py-2"
          placeholder="Acme, Inc."
        />
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-neutral-900 px-3 py-2 text-white disabled:opacity-50"
        >
          {submitting ? "Creating..." : "Continue"}
        </button>
        {err && <p className="text-sm text-red-600">{err}</p>}
      </form>
    </main>
  );
}
