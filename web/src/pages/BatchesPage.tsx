import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, Layers, RefreshCw } from "lucide-react";
import { useState } from "react";
import { EmptyState } from "../components/EmptyState";
import { batchesApi } from "../lib/api";
import { cn } from "../lib/utils";
import type { Batch } from "../types";

function statusBar(batch: Batch) {
  const total = batch.total || 1;
  const segments: { status: string; count: number; tone: string }[] = [
    { status: "delivered", count: batch.counts?.delivered || 0, tone: "bg-emerald-500" },
    { status: "undelivered", count: batch.counts?.undelivered || 0, tone: "bg-amber-500" },
    { status: "failed", count: batch.counts?.failed || 0, tone: "bg-red-500" },
  ];
  return (
    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted flex" aria-hidden="true">
      {segments
        .filter((segment) => segment.count > 0)
        .map((segment) => (
          <div
            key={segment.status}
            className={cn("h-full", segment.tone)}
            style={{ width: `${(segment.count / total) * 100}%` }}
          />
        ))}
    </div>
  );
}

function BatchDetail({ batch, onBack }: { batch: Batch; onBack: () => void }) {
  const rejected = batch.rejected || [];
  const statuses = Object.keys(batch.counts || {}).sort();
  return (
    <div className="space-y-4">
      <button
        type="button"
        onClick={onBack}
        className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="w-3.5 h-3.5" />
        Back to batches
      </button>

      <div className="rounded-xl bg-card border border-border shadow-xs p-5 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-mono text-xs text-foreground break-all">{batch.id}</span>
          <span className="px-2 py-0.5 rounded-full text-[11px] font-medium font-mono bg-muted text-muted-foreground">
            {batch.provider}
          </span>
          <span className="ml-auto font-mono text-[11px] text-muted-foreground tabular-nums">
            {batch.total} recipients
          </span>
        </div>
        {statusBar(batch)}
        <p className="text-[11px] text-muted-foreground">
          {batch.channel} · created {new Date(batch.created_at).toLocaleString()}
        </p>
      </div>

      <div className="rounded-xl bg-card border border-border shadow-xs p-5 space-y-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Status counts
        </h3>
        {statuses.length === 0 ? (
          <p className="text-[11px] text-muted-foreground">No status counts yet.</p>
        ) : (
          <dl className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {statuses.map((status) => (
              <div key={status} className="rounded-lg bg-muted/40 border border-border p-3">
                <dt className="text-[11px] text-muted-foreground">{status}</dt>
                <dd className="text-lg font-bold text-foreground font-mono tabular-nums">
                  {batch.counts?.[status] || 0}
                </dd>
              </div>
            ))}
          </dl>
        )}
      </div>

      <div className="rounded-xl bg-card border border-border shadow-xs p-5 space-y-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Rejected recipients ({rejected.length})
        </h3>
        {rejected.length === 0 ? (
          <p className="text-[11px] text-muted-foreground">
            Every recipient was accepted into the batch.
          </p>
        ) : (
          <ul className="divide-y divide-border rounded-lg border border-border overflow-hidden">
            {rejected.map((entry) => (
              <li key={entry.to} className="px-4 py-2.5 flex flex-wrap items-center gap-2">
                <span className="font-mono text-xs text-foreground">{entry.to}</span>
                <span className="ml-auto px-2 py-0.5 rounded-full text-[11px] font-medium font-mono bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300">
                  {entry.code}
                </span>
                <p className="w-full text-[11px] text-muted-foreground">{entry.message}</p>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

export function BatchesPage() {
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const { data, isLoading, isError, refetch, isRefetching } = useQuery({
    queryKey: ["batches"],
    queryFn: () => batchesApi.list({ limit: 50 }),
    enabled: selectedId === null,
  });

  const { data: selectedBatch, isLoading: selectedLoading } = useQuery({
    queryKey: ["batches", selectedId],
    queryFn: () => batchesApi.get(selectedId as string),
    enabled: selectedId !== null,
  });

  const batches = data?.batches || [];

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-foreground tracking-tight">
            Batches & Bulk Campaigns
          </h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Monitor bulk message dispatches and delivery progress across multiple recipients
          </p>
        </div>
        {selectedId === null && (
          <button
            type="button"
            onClick={() => refetch()}
            disabled={isRefetching}
            className="p-2 rounded-md border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted transition-colors self-start"
            title="Refresh"
            aria-label="Refresh batches"
          >
            <RefreshCw className={cn("w-4 h-4", isRefetching && "animate-spin")} />
          </button>
        )}
      </div>

      {selectedId !== null ? (
        selectedLoading || !selectedBatch ? (
          <p className="text-xs text-muted-foreground">Loading batch…</p>
        ) : (
          <BatchDetail batch={selectedBatch} onBack={() => setSelectedId(null)} />
        )
      ) : isLoading ? (
        <p className="text-xs text-muted-foreground">Loading batches…</p>
      ) : isError ? (
        <EmptyState
          icon={Layers}
          title="Could Not Load Batches"
          description="The batches API returned an error. Is the mocksms server running?"
          actionLabel="Retry"
          onAction={() => refetch()}
        />
      ) : batches.length === 0 ? (
        <EmptyState
          icon={Layers}
          title="No Message Batches"
          description="Batch dispatches from Twilio, Termii, or the native batch API will be tracked here with live status aggregation."
        />
      ) : (
        <div className="rounded-xl bg-card border border-border shadow-xs overflow-hidden">
          <ul className="divide-y divide-border">
            {batches.map((batch) => (
              <li key={batch.id}>
                <button
                  type="button"
                  onClick={() => setSelectedId(batch.id)}
                  className="w-full px-4 py-3 text-left hover:bg-muted/60 transition-colors space-y-2"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-mono text-xs text-foreground break-all">{batch.id}</span>
                    <span className="px-2 py-0.5 rounded-full text-[11px] font-medium font-mono bg-muted text-muted-foreground">
                      {batch.provider}
                    </span>
                    <span className="ml-auto font-mono text-[11px] text-muted-foreground tabular-nums">
                      {batch.total} recipients
                    </span>
                  </div>
                  {statusBar(batch)}
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
