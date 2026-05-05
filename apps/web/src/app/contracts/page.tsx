import { redirect } from "next/navigation";

import { getMe } from "@/lib/server-session";

export const dynamic = "force-dynamic";

export default async function Contracts() {
  const me = await getMe();
  if (!me) redirect("/signin");
  if (me.memberships.length === 0) redirect("/onboarding/organization");

  const orgName =
    me.current_organization_name ?? me.memberships[0].organization_name;

  return (
    <main className="flex min-h-screen flex-col gap-6 p-8">
      <header className="flex items-center justify-between border-b border-neutral-200 pb-3">
        <div>
          <h1 className="text-xl font-semibold">{orgName}</h1>
          <p className="text-xs text-neutral-500">{me.user_email}</p>
        </div>
        <form action="/auth/logout" method="post">
          <button className="text-sm text-neutral-500 underline">Sign out</button>
        </form>
      </header>

      <section>
        <h2 className="text-lg font-medium">Contracts</h2>
        <p className="mt-2 text-sm text-neutral-500">
          No contracts yet. Phase 1 story 2 (intake form) lights this up.
        </p>
      </section>
    </main>
  );
}
