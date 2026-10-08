import { AlertTriangle, CheckCircle2, Clock, CornerDownLeft, Send, XCircle } from "lucide-react";
import type React from "react";
import { cn } from "../lib/utils";
import type { MessageStatus } from "../types";

interface StatusChipProps {
  status: MessageStatus;
  className?: string;
  isSimulated?: boolean;
}

export const StatusChip: React.FC<StatusChipProps> = ({
  status,
  className,
  isSimulated = true,
}) => {
  switch (status) {
    case "queued":
      return (
        <span
          className={cn(
            "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300",
            className
          )}
        >
          <Clock className="w-3.5 h-3.5 text-neutral-500" aria-hidden="true" />
          <span>Queued</span>
        </span>
      );

    case "sent":
      return (
        <span
          className={cn(
            "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300",
            className
          )}
        >
          <Send className="w-3.5 h-3.5 text-blue-600 dark:text-blue-400" aria-hidden="true" />
          <span>Sent</span>
        </span>
      );

    case "delivered":
      return (
        <span
          className={cn(
            "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300",
            className
          )}
          title={isSimulated ? "Delivered (simulated locally)" : "Delivered"}
        >
          <CheckCircle2
            className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400"
            aria-hidden="true"
          />
          <span>Delivered{isSimulated ? " (simulated)" : ""}</span>
        </span>
      );

    case "undelivered":
      return (
        <span
          className={cn(
            "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-50 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300",
            className
          )}
        >
          <AlertTriangle
            className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400"
            aria-hidden="true"
          />
          <span>Undelivered</span>
        </span>
      );

    case "failed":
      return (
        <span
          className={cn(
            "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-800 dark:bg-red-950/40 dark:text-red-300",
            className
          )}
        >
          <XCircle className="w-3.5 h-3.5 text-red-600 dark:text-red-400" aria-hidden="true" />
          <span>Failed</span>
        </span>
      );

    case "received":
      return (
        <span
          className={cn(
            "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-teal-50 text-teal-800 dark:bg-teal-950/40 dark:text-teal-300",
            className
          )}
        >
          <CornerDownLeft
            className="w-3.5 h-3.5 text-teal-600 dark:text-teal-400"
            aria-hidden="true"
          />
          <span>Received</span>
        </span>
      );

    default:
      return (
        <span
          className={cn(
            "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-neutral-100 text-neutral-600",
            className
          )}
        >
          <span>{status}</span>
        </span>
      );
  }
};
