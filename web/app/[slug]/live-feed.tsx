"use client";

import { useEffect, useRef, useState, useCallback, Fragment } from "react";
import { RefreshCw, Wifi, WifiOff, Cable, RotateCcw } from "lucide-react";
import { type RelayRequest, type DeliveryAttempt, getWebSocketUrl, replayRequest } from "@/lib/api";
import { MethodBadge } from "@/components/method-badge";
import { DeliveryBadge } from "@/components/delivery-badge";
import { Button } from "@/components/ui/button";
import { timeAgo, fmtDate, cn } from "@/lib/utils";

type WsEvent =
  | { type: "request_captured"; data: RelayRequest }
  | { type: "delivery_attempt"; data: DeliveryAttempt };

type ReplayState = "idle" | "replaying" | "done_ok" | "done_err";

function decodeBody(b64: string, contentType: string): { text: string; isJson: boolean } {
  if (!b64) return { text: "", isJson: false };
  let text = "";
  try { text = atob(b64); } catch { return { text: "(binary / undecodable)", isJson: false }; }
  const isJson = contentType?.includes("json") || false;
  if (isJson) {
    try { return { text: JSON.stringify(JSON.parse(text), null, 2), isJson: true }; } catch { /**/ }
  }
  return { text, isJson: false };
}

function HeadersTable({ headers }: { headers: Record<string, string[]> }) {
  const entries = Object.entries(headers);
  if (!entries.length) return <p className="text-xs text-muted-foreground italic">No headers</p>;
  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-xs font-mono">
        <tbody>
          {entries.map(([key, vals]) => (
            <tr key={key} className="border-b border-border last:border-0 hover:bg-muted/20">
              <td className="px-3 py-2 text-muted-foreground w-1/3 align-top break-all">{key}</td>
              <td className="px-3 py-2 text-foreground break-all">{vals.join(", ")}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PipelineNode({ label, state }: { label: string; state: "waiting" | "active" | "done" }) {
  if (state === "active") {
    return (
      <div className="relative rounded overflow-hidden p-[1.5px]">
        {/* Spinning conic gradient — the "running border" */}
        <div
          className="absolute w-[200%] h-[200%] -top-1/2 -left-1/2 animate-[spin_1.2s_linear_infinite]"
          style={{
            background: "conic-gradient(from 0deg, transparent 0deg, var(--primary) 60deg, var(--primary) 90deg, transparent 150deg)",
          }}
        />
        <div className="relative z-10 rounded bg-card px-2.5 py-1 text-xs font-mono font-semibold text-primary">
          {label}
        </div>
      </div>
    );
  }
  if (state === "done") {
    return (
      <div className="rounded border border-muted-foreground/20 px-2.5 py-1 text-xs font-mono text-muted-foreground/50">
        ✓ {label}
      </div>
    );
  }
  return (
    <div className="rounded px-2.5 py-1 text-xs font-mono text-muted-foreground/25">
      {label}
    </div>
  );
}

// Self-advancing pipeline: mounts when replaying, advances one step at a time.
// Unmounts (and resets) when replaying ends.
function ReplayPipeline() {
  const [step, setStep] = useState(0); // 0=Relay, 1=CLI, 2=Local

  useEffect(() => {
    const t1 = setTimeout(() => setStep(1), 380);
    const t2 = setTimeout(() => setStep(2), 820);
    return () => { clearTimeout(t1); clearTimeout(t2); };
  }, []);

  const nodes = ["Relay", "CLI", "Local"];

  return (
    <div className="flex items-center gap-2 py-1.5">
      {nodes.map((label, i) => (
        <Fragment key={label}>
          <PipelineNode
            label={label}
            state={i < step ? "done" : i === step ? "active" : "waiting"}
          />
          {i < nodes.length - 1 && (
            <span className={cn(
              "text-xs transition-colors duration-300",
              i < step ? "text-muted-foreground/40" : "text-muted-foreground/15"
            )}>→</span>
          )}
        </Fragment>
      ))}
    </div>
  );
}

function RequestDetail({
  request,
  attempt,
  onReplay,
  replayState,
  justDelivered,
}: {
  request: RelayRequest;
  attempt: DeliveryAttempt | undefined;
  onReplay: () => void;
  replayState: ReplayState;
  justDelivered: boolean;
}) {
  const { text: bodyText, isJson } = decodeBody(request.body, request.content_type);
  const isReplaying = replayState === "replaying";
  const is2xx = attempt ? attempt.status_code >= 200 && attempt.status_code < 300 : false;

  return (
    <div className="space-y-5 p-4 overflow-y-auto h-full">
      <div className="flex items-start gap-3 flex-wrap">
        <MethodBadge method={request.method} />
        <code className="text-sm font-mono text-foreground break-all">{request.path}</code>
      </div>

      {/* Body — shown first so the user can immediately see what they're replaying */}
      {bodyText && (
        <div className="space-y-1.5">
          <div className="flex items-center gap-2">
            <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Body</h3>
            {request.content_type && (
              <span className="text-[10px] font-mono text-muted-foreground/60">{request.content_type}</span>
            )}
          </div>
          <pre className={cn(
            "rounded-lg border border-border bg-muted/20 p-3 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-all leading-relaxed",
            isJson ? "text-green-300/90" : "text-foreground"
          )}>{bodyText}</pre>
        </div>
      )}

      {/* Delivery card */}
      <div className={cn(
        "rounded-lg border p-3 space-y-2 transition-all duration-500",
        justDelivered && attempt?.delivered  && "border-green-500/50 bg-green-500/5",
        justDelivered && !attempt?.delivered && "border-red-500/50 bg-red-500/5",
        !justDelivered && "border-border bg-muted/20",
      )}>
        <div className="flex items-center justify-between">
          <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Delivery</p>
          <Button variant="outline" size="xs" onClick={onReplay} disabled={isReplaying} title="Replay">
            <RotateCcw className={cn("size-3", isReplaying && "animate-spin")} />
            {isReplaying ? "Replaying…" : "Replay"}
          </Button>
        </div>

        {isReplaying && <ReplayPipeline />}

        {replayState === "done_err" && (
          <p className="text-xs text-destructive">No response — is the CLI connected?</p>
        )}

        {attempt && !isReplaying ? (
          <div className="space-y-1 text-xs">
            <div className="flex items-center gap-2">
              <DeliveryBadge attempt={attempt} />
              <span className={cn("font-medium", is2xx ? "text-green-400" : "text-red-400")}>
                {attempt.delivered ? "Delivered" : "Failed"}
              </span>
              {attempt.is_replay && (
                <span className="rounded px-1.5 py-0.5 bg-primary/10 text-primary ring-1 ring-inset ring-primary/20 text-[10px]">replay</span>
              )}
            </div>
            {attempt.attempted_at && (
              <p className="text-muted-foreground">Attempted: <span className="text-foreground">{fmtDate(attempt.attempted_at)}</span></p>
            )}
            {attempt.error && <p className="text-destructive font-mono break-all">{attempt.error}</p>}
          </div>
        ) : !isReplaying && (
          <p className="text-xs text-muted-foreground italic">No delivery attempt yet — start the CLI to forward requests.</p>
        )}
      </div>

      {/* Response from local server */}
      {attempt?.response_body && (
        <div className="space-y-2">
          <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground flex items-center gap-2">
            Local Response
            {attempt.status_code > 0 && (
              <span className={cn(
                "rounded px-1.5 py-0.5 text-[10px] font-mono font-semibold",
                attempt.delivered ? "bg-green-500/10 text-green-400" : "bg-red-500/10 text-red-400"
              )}>
                {attempt.status_code}
              </span>
            )}
          </h3>
          {attempt.response_headers && Object.keys(attempt.response_headers).length > 0 && (
            <HeadersTable headers={attempt.response_headers} />
          )}
          {(() => {
            let display = "";
            let isJson = false;
            try {
              const raw = atob(attempt.response_body!);
              display = JSON.stringify(JSON.parse(raw), null, 2);
              isJson = true;
            } catch {
              try { display = atob(attempt.response_body!); } catch { display = "(binary)"; }
            }
            return (
              <pre className={cn(
                "rounded-lg border border-border bg-muted/20 p-3 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-all leading-relaxed",
                isJson ? "text-green-300/90" : "text-foreground"
              )}>
                {display || <span className="italic text-muted-foreground">(empty body)</span>}
              </pre>
            );
          })()}
        </div>
      )}

      {Object.keys(request.query_params ?? {}).length > 0 && (
        <div className="space-y-2">
          <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Query Parameters</h3>
          <div className="overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-xs font-mono">
              <tbody>
                {Object.entries(request.query_params).map(([key, vals]) => (
                  <tr key={key} className="border-b border-border last:border-0 hover:bg-muted/20">
                    <td className="px-3 py-2 text-muted-foreground w-1/3 align-top">{key}</td>
                    <td className="px-3 py-2 text-foreground break-all">{vals.join(", ")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="space-y-2">
        <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Headers</h3>
        <HeadersTable headers={request.headers ?? {}} />
      </div>

    </div>
  );
}

function EmptyDetail() {
  return (
    <div className="flex flex-col items-center justify-center h-full text-center p-8 space-y-3">
      <Cable className="size-10 text-muted-foreground/30" />
      <p className="text-sm text-muted-foreground">Select a request to inspect it</p>
    </div>
  );
}

function EmptyFeed() {
  return (
    <div className="flex flex-col items-center justify-center h-full text-center p-8 space-y-3">
      <RefreshCw className="size-8 text-muted-foreground/30" />
      <div className="space-y-1">
        <p className="text-sm font-medium text-muted-foreground">Waiting for webhooks…</p>
        <p className="text-xs text-muted-foreground/60">Point your webhook provider at the endpoint URL above.</p>
      </div>
    </div>
  );
}

function ForwardingBadge() {
  return (
    <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium bg-primary/10 text-primary ring-1 ring-inset ring-primary/20">
      <span className="animate-pulse size-1.5 rounded-full bg-primary inline-block" />
      Forwarding
    </span>
  );
}

export function LiveFeed({
  slug,
  initialRequests,
  initialAttempts,
}: {
  slug: string;
  initialRequests: RelayRequest[];
  initialAttempts: Record<string, DeliveryAttempt>;
}) {
  const [requests, setRequests]     = useState<RelayRequest[]>(initialRequests);
  const [attempts, setAttempts]     = useState<Record<string, DeliveryAttempt>>(initialAttempts);
  const [selectedId, setSelectedId] = useState<string | null>(initialRequests[0]?.id ?? null);
  const [connected, setConnected]   = useState(false);

  // Ref-backed so WS handler always sees current values without stale closures
  const replayStatesRef = useRef<Record<string, ReplayState>>({});
  const [replayStates, _setReplayStates] = useState<Record<string, ReplayState>>({});
  const setReplayState = useCallback((id: string, s: ReplayState) => {
    replayStatesRef.current = { ...replayStatesRef.current, [id]: s };
    _setReplayStates({ ...replayStatesRef.current });
  }, []);

  const [flashIds, setFlashIds] = useState<Set<string>>(new Set());
  const flash = useCallback((id: string) => {
    setFlashIds(p => new Set(p).add(id));
    setTimeout(() => setFlashIds(p => { const s = new Set(p); s.delete(id); return s; }), 2000);
  }, []);

  // Per-request timeout handles: auto-clear if WS event never arrives
  const timeoutHandles = useRef<Record<string, ReturnType<typeof setTimeout>>>({});

  useEffect(() => {
    const url = getWebSocketUrl(slug);
    function connect() {
      const ws = new WebSocket(url);
      ws.onopen  = () => setConnected(true);
      ws.onclose = () => { setConnected(false); setTimeout(connect, 2000); };
      ws.onerror = () => ws.close();
      ws.onmessage = (evt) => {
        try {
          const msg: WsEvent = JSON.parse(evt.data);
          if (msg.type === "request_captured") {
            setRequests(p => [msg.data, ...p]);
            setSelectedId(p => p ?? msg.data.id);
          } else if (msg.type === "delivery_attempt") {
            setAttempts(p => ({ ...p, [msg.data.request_id]: msg.data }));
            const rid = msg.data.request_id;
            const cur = replayStatesRef.current[rid];
            if (cur === "replaying" || cur === "done_err") {
              clearTimeout(timeoutHandles.current[rid]);
              delete timeoutHandles.current[rid];
              replayStatesRef.current = { ...replayStatesRef.current, [rid]: "done_ok" };
              _setReplayStates({ ...replayStatesRef.current });
              flash(rid);
              setTimeout(() => {
                replayStatesRef.current = { ...replayStatesRef.current, [rid]: "idle" };
                _setReplayStates({ ...replayStatesRef.current });
              }, 2200);
            }
          }
        } catch { /**/ }
      };
    }
    connect();
    return () => { /* ws is closed by onclose → reconnect; cleaned up on unmount by the ref */ };
  }, [slug, flash]);

  const handleReplay = useCallback(async () => {
    if (!selectedId) return;
    if (replayStatesRef.current[selectedId] === "replaying") return;

    setReplayState(selectedId, "replaying");

    // The server's forwarder always fires a delivery_attempt WS event within 30s
    // (success or CLI timeout). Safety net is 35s — only fires if WS itself died.
    const handle = setTimeout(() => {
      if (replayStatesRef.current[selectedId] === "replaying") {
        setReplayState(selectedId, "done_err");
        setTimeout(() => setReplayState(selectedId, "idle"), 4000);
      }
    }, 35_000);
    timeoutHandles.current[selectedId] = handle;

    try {
      await replayRequest(slug, selectedId);
    } catch {
      // Stay in "replaying" — if the server got it, the WS event will resolve it.
    }
  }, [slug, selectedId, setReplayState]);

  const selectedRequest = requests.find(r => r.id === selectedId);

  return (
    <div className="flex-1 flex flex-col">
      <div className="mb-3 flex items-center gap-2">
        <span className={cn(
          "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium",
          connected
            ? "bg-green-500/10 text-green-400 ring-1 ring-inset ring-green-500/20"
            : "bg-muted/50 text-muted-foreground ring-1 ring-inset ring-border"
        )}>
          {connected ? <Wifi className="size-3" /> : <WifiOff className="size-3" />}
          {connected ? "Live" : "Connecting…"}
        </span>
        <span className="text-xs text-muted-foreground">
          {requests.length} request{requests.length !== 1 ? "s" : ""}
        </span>
      </div>

      <div className="flex-1 grid grid-cols-[320px_1fr] gap-4 min-h-0" style={{ height: "calc(100vh - 180px)" }}>
        <div className="rounded-xl border border-border bg-card overflow-hidden flex flex-col">
          <div className="border-b border-border px-3 py-2.5 flex items-center gap-2">
            <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Requests</span>
            {requests.length > 0 && (
              <span className="ml-auto rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">{requests.length}</span>
            )}
          </div>
          <div className="flex-1 overflow-y-auto">
            {requests.length === 0 ? <EmptyFeed /> : (
              <ul>
                {requests.map(req => {
                  const attempt = attempts[req.id];
                  const isSelected = req.id === selectedId;
                  const forwarding = replayStates[req.id] === "replaying";
                  return (
                    <li key={req.id}>
                      <button
                        onClick={() => setSelectedId(req.id)}
                        className={cn(
                          "w-full text-left px-3 py-3 border-b border-border last:border-0 transition-colors hover:bg-muted/30",
                          isSelected && "bg-primary/10 border-l-2 border-l-primary",
                          flashIds.has(req.id) && "bg-primary/5",
                        )}
                      >
                        <div className="flex items-center gap-2 mb-1">
                          <MethodBadge method={req.method} />
                          <span className="flex-1 truncate text-xs font-mono text-foreground">{req.path || "/"}</span>
                        </div>
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-[10px] text-muted-foreground">{timeAgo(req.received_at)}</span>
                          {forwarding ? <ForwardingBadge /> : <DeliveryBadge attempt={attempt} />}
                        </div>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card overflow-hidden flex flex-col">
          {selectedRequest ? (
            <RequestDetail
              request={selectedRequest}
              attempt={attempts[selectedRequest.id]}
              onReplay={handleReplay}
              replayState={replayStates[selectedRequest.id] ?? "idle"}
              justDelivered={flashIds.has(selectedRequest.id)}
            />
          ) : <EmptyDetail />}
        </div>
      </div>
    </div>
  );
}
