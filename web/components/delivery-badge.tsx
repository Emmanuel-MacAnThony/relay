import { cn } from "@/lib/utils";
import type { DeliveryAttempt } from "@/lib/api";

export function DeliveryBadge({
  attempt,
}: {
  attempt: DeliveryAttempt | undefined;
}) {
  if (!attempt) {
    return (
      <span className="inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium ring-1 ring-inset bg-muted/50 text-muted-foreground ring-border">
        Pending
      </span>
    );
  }

  const is2xx = attempt.status_code >= 200 && attempt.status_code < 300;
  const hasCode = attempt.status_code > 0;

  if (!attempt.delivered || !is2xx) {
    return (
      <span
        className={cn(
          "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium ring-1 ring-inset font-mono",
          "bg-red-500/15 text-red-400 ring-red-500/20"
        )}
      >
        {hasCode ? attempt.status_code : "ERR"}
      </span>
    );
  }

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium ring-1 ring-inset font-mono",
        "bg-green-500/15 text-green-400 ring-green-500/20"
      )}
    >
      {attempt.status_code}
    </span>
  );
}
