import Link from "next/link";
import { redirect } from "next/navigation";
import { cookies } from "next/headers";

import { client } from "@/api/client";
import { getMe } from "@/lib/server-session";

export const dynamic = "force-dynamic";

export default async function Contracts() {
  const me = await getMe();
  if (!me) redirect("/signin");
  if (me.memberships.length === 0) redirect("/onboarding/organization");

  const session = (await cookies()).get("pactline_session")?.value ?? "";
  const { data } = await client.GET("/contracts", {
    headers: { Authorization: `Bearer ${session}` },
  });
  const contracts = data?.contracts ?? [];

  const orgName =
    me.current_organization_name ?? me.memberships[0].organization_name;

  return (
    <main className="flex min-h-screen flex-col gap-6 p-8">
      <header className="flex items-center justify-between border-b border-neutral-200 pb-3">
        <div>
          <h1 className="text-xl font-semibold">{orgName}</h1>
          <p className="text-xs text-neutral-500">{me.user_email}</p>
        </div>
        <div className="flex gap-3">
          <Link
            href="/contracts/new"
            className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white"
          >
            New contract
          </Link>
          <form action="/auth/logout" method="post">
            <button className="text-sm text-neutral-500 underline">
              Sign out
            </button>
          </form>
        </div>
      </header>

      <section>
        <h2 className="text-lg font-medium">Contracts</h2>
        {contracts.length === 0 ? (
          <p className="mt-2 text-sm text-neutral-500">
            No contracts yet. Click &quot;New contract&quot; to upload one.
          </p>
        ) : (
          <table className="mt-4 w-full text-sm">
            <thead className="text-left text-neutral-500">
              <tr>
                <th className="pb-2">Title</th>
                <th className="pb-2">Status</th>
                <th className="pb-2">Created</th>
              </tr>
            </thead>
            <tbody>
              {contracts.map((c) => (
                <tr key={c.id} className="border-t border-neutral-100">
                  <td className="py-2">
                    <Link
                      href={`/contracts/${c.id}`}
                      className="text-blue-700 underline"
                    >
                      {c.title}
                    </Link>
                  </td>
                  <td className="py-2">
                    <StatusBadge status={c.status} />
                  </td>
                  <td className="py-2 text-neutral-500">
                    {new Date(c.created_at).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </main>
  );
}

function StatusBadge({ status }: { status: string }) {
  const tone =
    status === "ready_for_review"
      ? "bg-green-100 text-green-800"
      : status === "parsing" || status === "intake"
        ? "bg-amber-100 text-amber-800"
        : "bg-neutral-100 text-neutral-800";
  return (
    <span className={`rounded-md px-2 py-0.5 text-xs ${tone}`}>{status}</span>
  );
}
