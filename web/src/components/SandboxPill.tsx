import { Info, ShieldCheck, X } from "lucide-react";
import type React from "react";
import { useEffect, useRef, useState } from "react";
import { cn } from "../lib/utils";

export const SandboxPill: React.FC<{ className?: string }> = ({ className }) => {
  const [open, setOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (popoverRef.current && !popoverRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    if (open) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [open]);

  return (
    <div className={cn("relative inline-block", className)} ref={popoverRef}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full border border-dashed border-ember-600 dark:border-ember-500 bg-ember-50/70 dark:bg-ember-950/40 text-ember-700 dark:text-ember-300 text-xs font-medium hover:bg-ember-100/80 transition-colors cursor-pointer select-none"
        aria-expanded={open}
        aria-haspopup="dialog"
      >
        <ShieldCheck className="w-4 h-4 text-ember-600 dark:text-ember-400" aria-hidden="true" />
        <span>Sandbox: nothing is delivered</span>
      </button>

      {open && (
        <div
          aria-label="Sandbox information"
          className="absolute right-0 top-full mt-2 w-80 p-4 rounded-lg bg-card text-card-foreground border border-border shadow-lg z-50 animate-in fade-in zoom-in-95 duration-150"
        >
          <div className="flex items-start justify-between mb-2">
            <div className="flex items-center gap-1.5 font-semibold text-sm text-foreground">
              <Info className="w-4 h-4 text-ember-500" />
              <span>mocksms Sandbox Mode</span>
            </div>
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="p-1 -mr-1 rounded hover:bg-muted text-muted-foreground transition-colors"
              aria-label="Close popover"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>

          <p className="text-xs text-muted-foreground leading-relaxed mb-3">
            mocksms operates completely locally on your machine. All SMS, OTPs, and emails are
            intercepted, validated, and stored in SQLite. No messages are ever delivered over real
            carrier or telecom networks.
          </p>

          <div className="space-y-1.5 text-xs bg-muted/50 p-2.5 rounded-md border border-border/50">
            <div className="flex items-center gap-2">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
              <span className="text-foreground font-medium">Safe for testing:</span>
              <span className="text-muted-foreground">$0 provider cost</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
              <span className="text-foreground font-medium">Deterministic:</span>
              <span className="text-muted-foreground">Reproducible webhooks</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
