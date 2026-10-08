import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, RefreshCw, Send, Terminal } from "lucide-react";
import { useState } from "react";
import { EmptyState } from "../components/EmptyState";
import { inspectorApi } from "../lib/api";
import { cn } from "../lib/utils";
import type { RequestLog } from "../types";

function statusTone(status: number): string {
  if (status >= 500) {
    return "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300";
  }
  if (status >= 400) {
    return "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300";
  }
  return "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300";
}

function formatBody(body: string | undefined): string {
  if (!body) {
    return "(empty)";
  }
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body;
  }
}

function HeaderList({ headers }: { headers: Record<string, string> }) {
  const entries = Object.entries(headers || {});
  if (entries.length === 0) {
    return <p className="text-[11px] text-muted-foreground">(no headers)</p>;
  }
  return (
    <dl className="space-y-1">
      {entries.map(([name, value]) => (
        <div key={name} className="flex gap-2 text-[11px] leading-relaxed">
          <dt className="font-mono font-semibold text-foreground shrink-0">{name}:</dt>
          <dd className="font-mono text-muted-foreground break-all">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function RequestDetail({ log, onBack }: { log: RequestLog; onBack: () => void }) {
  return (
    <div className="space-y-4">
      <button
        type="button"
        onClick={onBack}
        className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="w-3.5 h-3.5" />
        Back to requests
      </button>

      <div className="rounded-xl bg-card border border-border shadow-xs p-5 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="px-2 py-0.5 rounded-md bg-muted font-mono text-[11px] font-semibold text-foreground">
            {log.method}
          </span>
          <span className="font-mono text-xs text-foreground break-all">{log.path}</span>
          <span
            className={cn(
              "ml-auto px-2 py-0.5 rounded-full text-[11px] font-medium font-mono",
              statusTone(log.response_status)
            )}
          >
            {log.response_status}
          </span>
        </div>
        <p className="text-[11px] text-muted-foreground font-mono">
          {log.adapter} · {log.duration_ms} ms · {new Date(log.created_at).toLocaleString()}
        </p>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <div className="rounded-xl bg-card border border-border shadow-xs p-5 space-y-3">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Request
          </h3>
          <HeaderList headers={log.request_headers} />
          <pre className="p-3 rounded-md bg-muted font-mono text-[11px] leading-relaxed overflow-auto max-h-96 whitespace-pre-wrap break-all">
            {formatBody(log.request_body)}
          </pre>
        </div>
        <div className="rounded-xl bg-card border border-border shadow-xs p-5 space-y-3">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Response · {log.response_status}
          </h3>
          <pre className="p-3 rounded-md bg-muted font-mono text-[11px] leading-relaxed overflow-auto max-h-96 whitespace-pre-wrap break-all">
            {formatBody(log.response_body)}
          </pre>
        </div>
      </div>
    </div>
  );
}

export function InspectorPage() {
  const [activeTab, setActiveTab] = useState<"requests" | "webhooks">("requests");
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const {
    data: requestsData,
    isLoading: requestsLoading,
    isError: requestsError,
    refetch: refetchRequests,
    isRefetching: requestsRefetching,
  } = useQuery({
    queryKey: ["request-logs"],
    queryFn: () => inspectorApi.listRequests({ limit: 50 }),
    enabled: activeTab === "requests" && selectedId === null,
  });

  const { data: selectedLog, isLoading: selectedLoading } = useQuery({
    queryKey: ["request-logs", selectedId],
    queryFn: () => inspectorApi.getRequest(selectedId as string),
    enabled: selectedId !== null,
  });

  const {
    data: webhooksData,
    isLoading: webhooksLoading,
    isError: webhooksError,
    refetch: refetchWebhooks,
  } = useQuery({
    queryKey: ["webhook-deliveries"],
    queryFn: () => inspectorApi.listWebhooks({ limit: 50 }),
    enabled: activeTab === "webhooks",
  });

  const logs = requestsData?.logs || [];
  const deliveries = webhooksData?.deliveries || [];

  return (
    <div className="space-y-6">
      {/* Header with Tabs */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-foreground tracking-tight">
            Inspector & Observability
          </h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Inspect incoming HTTP requests, headers, and outgoing webhook delivery attempts
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => (activeTab === "requests" ? refetchRequests() : refetchWebhooks())}
            disabled={requestsRefetching}
            className="p-2 rounded-md border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            title="Refresh"
            aria-label="Refresh"
          >
            <RefreshCw className={cn("w-4 h-4", requestsRefetching && "animate-spin")} />
          </button>
          <div className="flex items-center gap-1 p-1 bg-muted rounded-lg text-xs font-medium self-start">
            <button
              type="button"
              onClick={() => {
                setActiveTab("requests");
                setSelectedId(null);
              }}
              className={cn(
                "flex items-center gap-1.5 px-3 py-1.5 rounded-md transition-all",
                activeTab === "requests"
                  ? "bg-card text-foreground shadow-sm font-semibold"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <Terminal className="w-3.5 h-3.5 text-ember-600" />
              <span>HTTP Requests</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("webhooks")}
              className={cn(
                "flex items-center gap-1.5 px-3 py-1.5 rounded-md transition-all",
                activeTab === "webhooks"
                  ? "bg-card text-foreground shadow-sm font-semibold"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <Send className="w-3.5 h-3.5 text-blue-600" />
              <span>Webhooks Log</span>
            </button>
          </div>
        </div>
      </div>

      {activeTab === "requests" ? (
        selectedId !== null ? (
          selectedLoading || !selectedLog ? (
            <p className="text-xs text-muted-foreground">Loading request…</p>
          ) : (
            <RequestDetail log={selectedLog} onBack={() => setSelectedId(null)} />
          )
        ) : requestsLoading ? (
          <p className="text-xs text-muted-foreground">Loading requests…</p>
        ) : requestsError ? (
          <EmptyState
            icon={Terminal}
            title="Could Not Load Requests"
            description="The request log API returned an error. Is the mocksms server running?"
            actionLabel="Retry"
            onAction={() => refetchRequests()}
          />
        ) : logs.length === 0 ? (
          <EmptyState
            icon={Terminal}
            title="No Request Logs"
            description="Inbound requests received by provider adapters will be logged here with headers and masked credentials."
          />
        ) : (
          <div className="rounded-xl bg-card border border-border shadow-xs overflow-hidden">
            <ul className="divide-y divide-border">
              {logs.map((log) => (
                <li key={log.id}>
                  <button
                    type="button"
                    onClick={() => setSelectedId(log.id)}
                    className="w-full flex flex-wrap items-center gap-2 px-4 py-3 text-left hover:bg-muted/60 transition-colors"
                  >
                    <span className="px-2 py-0.5 rounded-md bg-muted font-mono text-[11px] font-semibold text-foreground">
                      {log.method}
                    </span>
                    <span className="font-mono text-xs text-foreground break-all">{log.path}</span>
                    <span className="ml-auto flex items-center gap-2">
                      <span className="font-mono text-[11px] text-muted-foreground tabular-nums">
                        {log.duration_ms} ms
                      </span>
                      <span
                        className={cn(
                          "px-2 py-0.5 rounded-full text-[11px] font-medium font-mono",
                          statusTone(log.response_status)
                        )}
                      >
                        {log.response_status}
                      </span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )
      ) : webhooksLoading ? (
        <p className="text-xs text-muted-foreground">Loading webhook deliveries…</p>
      ) : webhooksError ? (
        <EmptyState
          icon={Send}
          title="Could Not Load Webhooks"
          description="The webhook log API returned an error. Is the mocksms server running?"
          actionLabel="Retry"
          onAction={() => refetchWebhooks()}
        />
      ) : deliveries.length === 0 ? (
        <EmptyState
          icon={Send}
          title="No Webhook Deliveries"
          description="Outgoing status callback webhooks triggered by message status transitions will appear here."
        />
      ) : (
        <div className="rounded-xl bg-card border border-border shadow-xs overflow-hidden">
          <ul className="divide-y divide-border">
            {deliveries.map((delivery) => (
              <li key={delivery.id} className="px-4 py-3 space-y-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-xs text-foreground break-all">
                    {delivery.url}
                  </span>
                  <span className="ml-auto px-2 py-0.5 rounded-full text-[11px] font-medium font-mono bg-muted text-muted-foreground">
                    {delivery.status}
                  </span>
                </div>
                <p className="text-[11px] text-muted-foreground font-mono">
                  {delivery.kind} · attempt {delivery.attempt} ·{" "}
                  {new Date(delivery.created_at).toLocaleString()}
                </p>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
