import { AlertTriangle, Check, CheckCircle2, Clock, Send, XCircle } from "lucide-react";
import type React from "react";
import { cn } from "../lib/utils";
import type { MessageStatus, StatusEvent } from "../types";

interface DeliveryTrackProps {
  status: MessageStatus;
  statusEvents?: StatusEvent[];
  createdAt: string;
  errorCode?: string;
  errorMessage?: string;
  className?: string;
}

export const DeliveryTrack: React.FC<DeliveryTrackProps> = ({
  status,
  statusEvents = [],
  createdAt,
  errorCode,
  errorMessage,
  className,
}) => {
  const isFailed = status === "failed";
  const isUndelivered = status === "undelivered";
  const isDelivered = status === "delivered";
  const isSent = status === "sent" || isDelivered || isUndelivered || isFailed;

  const getTimeFor = (st: MessageStatus) => {
    const ev = statusEvents.find((e) => e.status === st);
    if (ev) {
      return new Date(ev.at).toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      });
    }
    if (st === "queued") {
      return new Date(createdAt).toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      });
    }
    return null;
  };

  const steps = [
    {
      id: "queued",
      label: "Queued",
      completed: true,
      time: getTimeFor("queued"),
      icon: Clock,
    },
    {
      id: "sent",
      label: "Sent",
      completed: isSent,
      time: getTimeFor("sent"),
      icon: Send,
    },
    {
      id: "delivery",
      label: isFailed ? "Failed" : isUndelivered ? "Undelivered" : "Delivered (simulated)",
      completed: isDelivered || isUndelivered || isFailed,
      isCurrent: true,
      time: getTimeFor(status),
      icon: isFailed ? XCircle : isUndelivered ? AlertTriangle : CheckCircle2,
      isError: isFailed || isUndelivered,
    },
  ];

  return (
    <div className={cn("p-4 rounded-lg bg-card border border-border", className)}>
      <div className="flex items-center justify-between mb-4">
        <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          Delivery Lifecycle
        </span>
        <span className="text-xs text-muted-foreground font-mono">Local virtual timeline</span>
      </div>

      <div className="relative flex items-center justify-between">
        {/* Track connecting line */}
        <div className="absolute top-4 left-6 right-6 h-0.5 bg-neutral-200 dark:bg-neutral-800 -z-0" />

        {steps.map((step, idx) => {
          const Icon = step.icon;
          return (
            <div
              key={step.id}
              className="relative z-10 flex flex-col items-center flex-1 first:items-start last:items-end text-center"
            >
              <div
                className={cn(
                  "w-8 h-8 rounded-full flex items-center justify-center transition-colors",
                  step.isError
                    ? "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300 ring-4 ring-red-50 dark:ring-red-950/40"
                    : step.completed
                      ? "bg-emerald-600 text-white ring-4 ring-emerald-50 dark:ring-emerald-950/40"
                      : "bg-muted text-muted-foreground border border-border"
                )}
              >
                {step.completed && !step.isError && idx < 2 ? (
                  <Check className="w-4 h-4" />
                ) : (
                  <Icon className="w-4 h-4" />
                )}
              </div>

              <span
                className={cn(
                  "mt-2 text-xs font-medium",
                  step.isError
                    ? "text-red-600 dark:text-red-400 font-semibold"
                    : step.completed
                      ? "text-foreground font-semibold"
                      : "text-muted-foreground"
                )}
              >
                {step.label}
              </span>

              {step.time && (
                <span className="text-[11px] text-muted-foreground tabular-nums font-mono mt-0.5">
                  {step.time}
                </span>
              )}
            </div>
          );
        })}
      </div>

      {(errorCode || errorMessage) && (
        <div className="mt-4 p-2.5 rounded-md bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-900/60 text-xs text-red-800 dark:text-red-300 flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 shrink-0 text-red-600" />
          <div>
            <span className="font-semibold font-mono mr-1">[{errorCode}]:</span>
            <span>{errorMessage || "Simulation failure"}</span>
          </div>
        </div>
      )}
    </div>
  );
};
