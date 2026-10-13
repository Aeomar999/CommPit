import { useQuery } from "@tanstack/react-query";
import { Calculator, RefreshCw } from "lucide-react";
import { useState } from "react";
import { EmptyState } from "../components/EmptyState";
import { estimateApi } from "../lib/api";
import { cn } from "../lib/utils";
import type { ProviderEstimate } from "../types";

function money(value: number, currency: string): string {
  return `${currency} ${value.toFixed(2)}`;
}

function ProviderCard({ estimate }: { estimate: ProviderEstimate }) {
  return (
    <div className="rounded-xl bg-card border border-border shadow-xs p-5 sm:p-6 space-y-4">
      <div className="flex flex-wrap items-center gap-2 pb-3 border-b border-border">
        <span className="px-2 py-0.5 rounded-full text-[11px] font-medium font-mono bg-muted text-muted-foreground uppercase">
          {estimate.provider}
        </span>
        <span
          className="px-2 py-0.5 rounded-full text-[11px] font-medium bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
          title="Prices change; confirm rate cards before launch"
        >
          estimate only
        </span>
        <span className="ml-auto font-mono text-[11px] text-muted-foreground tabular-nums">
          prices as of {estimate.as_of}
        </span>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <div>
          <p className="text-[11px] text-muted-foreground">Monthly messages</p>
          <p className="text-xl font-bold text-foreground font-mono tabular-nums">
            {estimate.monthly_messages.toLocaleString()}
          </p>
        </div>
        <div>
          <p className="text-[11px] text-muted-foreground">Messaging cost</p>
          <p className="text-xl font-bold text-foreground font-mono tabular-nums">
            {money(estimate.total_cost, estimate.currency)}
          </p>
        </div>
        <div>
          <p className="text-[11px] text-muted-foreground">
            Verifications ({estimate.monthly_verifications.toLocaleString()})
          </p>
          <p className="text-xl font-bold text-foreground font-mono tabular-nums">
            {money(estimate.verify_cost, estimate.currency)}
          </p>
        </div>
      </div>

      {estimate.by_country.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            By country
          </h4>
          <ul className="divide-y divide-border rounded-lg border border-border overflow-hidden">
            {estimate.by_country.map((row) => (
              <li
                key={row.country || "unknown"}
                className="px-4 py-2.5 flex flex-wrap items-center gap-2"
              >
                <span className="font-mono text-xs font-semibold text-foreground">
                  {row.country || "unknown"}
                </span>
                <span className="font-mono text-[11px] text-muted-foreground tabular-nums">
                  {row.messages.toLocaleString()} msgs × {row.unit_price}
                </span>
                <span className="ml-auto font-mono text-xs font-semibold text-foreground tabular-nums">
                  {money(row.cost, estimate.currency)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {estimate.no_price_data.length > 0 && (
        <p className="text-[11px] text-muted-foreground">
          No price data for{" "}
          {estimate.no_price_data
            .map((row) => `${row.country || "unknown"} (${row.messages.toLocaleString()} msgs)`)
            .join(", ")}{" "}
          — excluded from the total, never guessed.
        </p>
      )}
    </div>
  );
}

export function EstimatePage() {
  const [volumeInput, setVolumeInput] = useState("10000");
  const [volume, setVolume] = useState(10000);

  const { data, isLoading, isError, refetch, isRefetching } = useQuery({
    queryKey: ["estimate", volume],
    queryFn: () => estimateApi.get(volume),
    enabled: volume > 0,
  });

  const providers = data?.providers || [];

  return (
    <div className="space-y-6 max-w-4xl">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-foreground tracking-tight">Go-Live Estimate</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Your observed traffic mix scaled to launch volume, priced per provider
          </p>
        </div>
        <button
          type="button"
          onClick={() => refetch()}
          disabled={isRefetching}
          className="p-2 rounded-md border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted transition-colors self-start"
          title="Refresh"
          aria-label="Refresh estimate"
        >
          <RefreshCw className={cn("w-4 h-4", isRefetching && "animate-spin")} />
        </button>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          const next = Number(volumeInput);
          if (Number.isFinite(next) && next > 0) {
            setVolume(Math.round(next));
          }
        }}
        className="rounded-xl bg-card border border-border shadow-xs p-5 sm:p-6 space-y-4"
      >
        <div className="flex items-center gap-2 pb-3 border-b border-border">
          <Calculator className="w-4 h-4 text-ember-600" />
          <h3 className="text-sm font-semibold text-foreground">Launch volume</h3>
        </div>
        <div className="space-y-1.5">
          <label htmlFor="estimate-volume" className="text-xs font-semibold text-foreground">
            Messages per month
          </label>
          <div className="flex flex-col sm:flex-row gap-3">
            <input
              id="estimate-volume"
              type="number"
              min={1}
              step={100}
              value={volumeInput}
              onChange={(e) => setVolumeInput(e.target.value)}
              className="flex-1 px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500 font-mono tabular-nums"
            />
            <button
              type="submit"
              className="inline-flex items-center justify-center gap-2 px-5 py-2 rounded-md bg-ember-500 hover:bg-ember-600 text-ember-950 font-semibold text-xs shadow-sm transition-all active:scale-[0.98]"
            >
              <Calculator className="w-4 h-4" />
              <span>Estimate</span>
            </button>
          </div>
          <p className="text-[11px] text-muted-foreground">
            Observed traffic mix
            {data
              ? ` (${data.observed_messages} messages, ${data.observed_verifications} verifications)`
              : ""}{" "}
            scales to this volume.
          </p>
        </div>
      </form>

      {isLoading ? (
        <p className="text-xs text-muted-foreground">Estimating…</p>
      ) : isError ? (
        <EmptyState
          icon={Calculator}
          title="Could Not Load Estimate"
          description="The estimate API returned an error. Is the mocksms server running?"
          actionLabel="Retry"
          onAction={() => refetch()}
        />
      ) : providers.length === 0 ? (
        <EmptyState
          icon={Calculator}
          title="No Priced Traffic Yet"
          description="Send some SMS through a priced provider (Twilio, Termii) first — the estimate scales your observed mix, and native or SMTP traffic has no rate card."
        />
      ) : (
        <div className="space-y-4">
          {providers.map((estimate) => (
            <ProviderCard key={estimate.provider} estimate={estimate} />
          ))}
        </div>
      )}
    </div>
  );
}
