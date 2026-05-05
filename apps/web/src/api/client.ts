import createClient from "openapi-fetch";

import type { paths } from "./types";

const baseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8000";

export const client = createClient<paths>({ baseUrl });
