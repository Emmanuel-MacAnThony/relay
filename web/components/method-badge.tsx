import { cn } from "@/lib/utils";

const METHOD_STYLES: Record<string, string> = {
  GET: "bg-blue-500/15 text-blue-400 ring-blue-500/20",
  POST: "bg-green-500/15 text-green-400 ring-green-500/20",
  PUT: "bg-yellow-500/15 text-yellow-400 ring-yellow-500/20",
  PATCH: "bg-orange-500/15 text-orange-400 ring-orange-500/20",
  DELETE: "bg-red-500/15 text-red-400 ring-red-500/20",
  HEAD: "bg-purple-500/15 text-purple-400 ring-purple-500/20",
  OPTIONS: "bg-gray-500/15 text-gray-400 ring-gray-500/20",
};

export function MethodBadge({ method }: { method: string }) {
  const upper = method.toUpperCase();
  const style = METHOD_STYLES[upper] ?? "bg-gray-500/15 text-gray-400 ring-gray-500/20";
  return (
    <span
      className={cn(
        "inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider ring-1 ring-inset font-mono",
        style
      )}
    >
      {upper}
    </span>
  );
}
