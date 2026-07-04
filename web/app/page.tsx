"use client";

import { useState, useTransition } from "react";
import { useRouter } from "next/navigation";
import { Cable, Plus, ArrowRight, Webhook } from "lucide-react";
import { Button } from "@/components/ui/button";
import { createEndpointAction } from "@/app/actions";

export default function HomePage() {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);
  const [slug, setSlug] = useState("");

  function handleCreate() {
    setError(null);
    startTransition(async () => {
      const result = await createEndpointAction();
      if (result?.error) {
        setError(result.error);
      }
    });
  }

  function handleGo(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = slug.trim();
    if (!trimmed) return;
    router.push(`/${trimmed}`);
  }

  return (
    <div className="min-h-screen bg-grid bg-background flex flex-col">
      {/* Header */}
      <header className="sticky top-0 z-10 border-b border-border bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex max-w-4xl items-center px-6 py-3.5">
          <div className="flex items-center gap-2.5">
            <Cable className="size-5 text-primary" />
            <span className="text-base font-semibold tracking-tight">
              Relay
            </span>
          </div>
        </div>
      </header>

      {/* Hero */}
      <main className="flex-1 flex flex-col items-center justify-center px-6 py-16">
        <div className="w-full max-w-md space-y-10">
          {/* Title block */}
          <div className="text-center space-y-3">
            <div className="inline-flex items-center justify-center rounded-2xl border border-border bg-card p-4 mb-2">
              <Webhook className="size-10 text-primary" />
            </div>
            <h1 className="text-3xl font-bold tracking-tight">
              Webhook Relay
            </h1>
            <p className="text-muted-foreground text-sm leading-relaxed">
              Capture webhooks from GitHub, Stripe, and more — and forward them
              to your local dev server in real time.
            </p>
          </div>

          {/* Create new endpoint */}
          <div className="rounded-xl border border-border bg-card p-6 space-y-4">
            <div className="space-y-1">
              <h2 className="text-sm font-semibold">New endpoint</h2>
              <p className="text-xs text-muted-foreground">
                Create a unique relay URL to point your webhook provider at.
              </p>
            </div>

            {error && (
              <div className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">
                {error}
              </div>
            )}

            <Button
              onClick={handleCreate}
              disabled={isPending}
              size="lg"
              className="w-full h-10 text-sm"
            >
              <Plus className="size-4" />
              {isPending ? "Creating…" : "Create new endpoint"}
            </Button>
          </div>

          {/* Go to existing */}
          <div className="rounded-xl border border-border bg-card p-6 space-y-4">
            <div className="space-y-1">
              <h2 className="text-sm font-semibold">Existing endpoint</h2>
              <p className="text-xs text-muted-foreground">
                Already have a slug? Jump straight to its dashboard.
              </p>
            </div>

            <form onSubmit={handleGo} className="flex gap-2">
              <input
                type="text"
                value={slug}
                onChange={(e) => setSlug(e.target.value)}
                placeholder="my-slug"
                className="flex-1 rounded-lg border border-border bg-input/30 px-3 py-2 text-sm font-mono placeholder:text-muted-foreground/50 focus:outline-none focus:ring-2 focus:ring-ring/50 focus:border-ring transition-all"
              />
              <Button type="submit" variant="outline" size="default">
                Go
                <ArrowRight className="size-3.5" />
              </Button>
            </form>
          </div>
        </div>
      </main>
    </div>
  );
}
