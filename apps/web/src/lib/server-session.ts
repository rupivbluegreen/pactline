import { cookies } from "next/headers";

import { client } from "@/api/client";

export async function getMe() {
  const cookieStore = await cookies();
  const session = cookieStore.get("pactline_session")?.value;
  if (!session) return null;

  const { data, error } = await client.GET("/me", {
    headers: { Authorization: `Bearer ${session}` },
  });
  if (error) return null;
  return data;
}
