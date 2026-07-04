import Link from "next/link";
import { Cable, ArrowLeft } from "lucide-react";
import { listRequests, listAttempts, type RelayRequest, type DeliveryAttempt } from "@/lib/api";
import { LiveFeed } from "./live-feed";
import { CopyButton } from "./copy-button";

export default async function SlugPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;

  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const webhookUrl = `${apiUrl}/hook/${slug}`;

  let initialRequests: RelayRequest[] = [];
  let initialAttempts: Record<string, DeliveryAttempt> = {};
  try {
    const [requests, attempts] = await Promise.all([
      listRequests(slug),
      listAttempts(slug),
    ]);
    initialRequests = requests;
    initialAttempts = Object.fromEntries(attempts.map((a) => [a.request_id, a]));
  } catch {
    // Show empty state — backend may not be running yet
  }

  return (
    <div className="min-h-screen bg-grid bg-background flex flex-col">
      <header className="sticky top-0 z-10 border-b border-border bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-3.5">
          <div className="flex items-center gap-3">
            <Link
              href="/"
              className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <ArrowLeft className="size-3.5" />
            </Link>
            <div className="w-px h-4 bg-border" />
            <div className="flex items-center gap-2">
              <Cable className="size-4 text-primary" />
              <span className="text-sm font-semibold tracking-tight">Relay</span>
            </div>
          </div>

          <div className="flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-3 py-1.5">
            <span className="text-xs text-muted-foreground">Endpoint:</span>
            <code className="text-xs font-mono text-foreground">{webhookUrl}</code>
            <CopyButton text={webhookUrl} />
          </div>
        </div>
      </header>

      <main className="flex-1 flex flex-col mx-auto w-full max-w-7xl px-6 py-4">
        <div className="mb-4 flex items-center gap-3">
          <span className="text-xs text-muted-foreground uppercase tracking-wider">Slug</span>
          <code className="rounded border border-border bg-muted/30 px-2 py-0.5 text-sm font-mono text-primary">
            {slug}
          </code>
        </div>

        <LiveFeed
          slug={slug}
          initialRequests={initialRequests}
          initialAttempts={initialAttempts}
        />
      </main>
    </div>
  );
}
