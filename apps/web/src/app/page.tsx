import { client } from "@/api/client";

export const dynamic = "force-dynamic";

export default async function Home() {
  const { data, error } = await client.GET("/hello", {
    params: { query: { name: "pactline" } },
  });

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 p-8">
      <h1 className="text-4xl font-semibold">Pactline</h1>
      <p className="text-sm text-neutral-500">
        Contract lifecycle platform built on durable workflows.
      </p>
      <div className="rounded-md border border-neutral-200 bg-neutral-50 px-4 py-2 font-mono text-sm">
        {error
          ? `api error: ${String(error)}`
          : `api says: ${data?.message ?? "(no message)"}`}
      </div>
    </main>
  );
}
