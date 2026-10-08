import { Layers } from "lucide-react";
import { EmptyState } from "../components/EmptyState";

export function BatchesPage() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-foreground tracking-tight">
            Batches & Bulk Campaigns
          </h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Monitor bulk message dispatches and delivery progress across multiple recipients
          </p>
        </div>
      </div>

      {/* Empty State */}
      <EmptyState
        icon={Layers}
        title="No Message Batches"
        description="Batch dispatches from Twilio, Termii, or the native batch API will be tracked here with live status aggregation."
        actionLabel="Refresh Batches"
        onAction={() => {}}
      />
    </div>
  );
}
