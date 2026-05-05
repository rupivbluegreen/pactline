import { redirect } from "next/navigation";
import { cookies } from "next/headers";

import { client } from "@/api/client";

export const dynamic = "force-dynamic";

export default async function Verify({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>;
}) {
  const { token } = await searchParams;
  if (!token) {
    redirect("/signin");
  }

  const { data, error } = await client.POST("/auth/magic-link/verify", {
    body: { token: token! },
  });
  if (error || !data) {
    redirect("/signin?error=invalid");
  }

  const cookieStore = await cookies();
  cookieStore.set("pactline_session", data!.session_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    expires: new Date(data!.expires_at),
    path: "/",
  });

  // Find out whether user already has an org.
  const meRes = await client.GET("/me", {
    headers: { Authorization: `Bearer ${data!.session_token}` },
  });
  if (meRes.data && meRes.data.memberships.length > 0) {
    redirect("/contracts");
  }
  redirect("/onboarding/organization");
}
