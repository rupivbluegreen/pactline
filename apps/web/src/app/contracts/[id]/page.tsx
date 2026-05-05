import { redirect } from "next/navigation";
import { cookies } from "next/headers";

import { client } from "@/api/client";
import { getMe } from "@/lib/server-session";
import { StatusPoller } from "./StatusPoller";

export const dynamic = "force-dynamic";

export default async function ContractDetail({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const me = await getMe();
  if (!me) redirect("/signin");

  const { id } = await params;
  const session = (await cookies()).get("pactline_session")?.value ?? "";
  const { data, error } = await client.GET("/contracts/{id}", {
    params: { path: { id } },
    headers: { Authorization: `Bearer ${session}` },
  });
  if (error || !data) redirect("/contracts");

  const docRes = await client.GET("/contracts/{id}/document", {
    params: { path: { id } },
    headers: { Authorization: `Bearer ${session}` },
  });
  const downloadURL = docRes.data?.url;

  return (
    <main className="flex min-h-screen flex-col gap-6 p-8">
      <StatusPoller status={data.status} />
      <header className="flex items-center justify-between border-b border-neutral-200 pb-3">
        <div>
          <h1 className="text-xl font-semibold">{data.title}</h1>
          <p className="text-xs text-neutral-500">
            {data.contract_type.toUpperCase()} · status: {data.status}
          </p>
        </div>
        {downloadURL && (
          <a
            href={downloadURL}
            className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm"
          >
            Download document
          </a>
        )}
      </header>

      <section>
        <h2 className="text-lg font-medium">Extracted fields</h2>
        {data.extracted_fields.length === 0 ? (
          <p className="mt-2 text-sm text-neutral-500">
            {data.status === "parsing" || data.status === "intake"
              ? "Extracting…"
              : "No fields extracted."}
          </p>
        ) : (
          <table className="mt-4 w-full text-sm">
            <thead className="text-left text-neutral-500">
              <tr>
                <th className="pb-2">Field</th>
                <th className="pb-2">Value</th>
                <th className="pb-2">Citation</th>
                <th className="pb-2">Model</th>
              </tr>
            </thead>
            <tbody>
              {data.extracted_fields.map((f) => (
                <tr
                  key={f.field_name}
                  className="border-t border-neutral-100 align-top"
                >
                  <td className="py-2 font-medium">{f.field_name}</td>
                  <td className="py-2">{f.value}</td>
                  <td className="py-2 text-neutral-500">
                    {f.page_or_paragraph} · [{f.span_start},{f.span_end})
                  </td>
                  <td className="py-2 text-neutral-500">
                    {f.model_id} · {f.prompt_version}
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
