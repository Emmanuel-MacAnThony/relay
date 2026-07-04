"use server";

import { redirect } from "next/navigation";
import { createEndpoint } from "@/lib/api";

export async function createEndpointAction(): Promise<{ error?: string }> {
  let slug: string;
  try {
    const endpoint = await createEndpoint();
    slug = endpoint.slug;
  } catch {
    return { error: "Could not reach the API server. Make sure the backend is running." };
  }
  redirect(`/${slug}`);
}
