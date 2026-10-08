import { Send, Terminal } from "lucide-react";
import { useState } from "react";
import { EmptyState } from "../components/EmptyState";
import { cn } from "../lib/utils";

export function InspectorPage() {
  const [activeTab, setActiveTab] = useState<"requests" | "webhooks">("requests");

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

        <div className="flex items-center gap-1 p-1 bg-muted rounded-lg text-xs font-medium self-start">
          <button
            type="button"
            onClick={() => setActiveTab("requests")}
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

      {activeTab === "requests" ? (
        <EmptyState
          icon={Terminal}
          title="No Request Logs"
          description="Inbound requests received by provider adapters will be logged here with headers and masked credentials."
        />
      ) : (
        <EmptyState
          icon={Send}
          title="No Webhook Deliveries"
          description="Outgoing status callback webhooks triggered by message status transitions will appear here."
        />
      )}
    </div>
  );
}
