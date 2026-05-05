import { redirect } from "next/navigation";

import { getMe } from "@/lib/server-session";

export const dynamic = "force-dynamic";

export default async function Home() {
  const me = await getMe();
  if (!me) redirect("/signin");
  if (me.memberships.length === 0) redirect("/onboarding/organization");
  redirect("/contracts");
}
