import { Inbox, type LucideIcon, Plus } from "lucide-react";
import type React from "react";
import { cn } from "../lib/utils";

interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
  secondaryActionLabel?: string;
  onSecondaryAction?: () => void;
  className?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon: Icon = Inbox,
  title,
  description,
  actionLabel,
  onAction,
  secondaryActionLabel,
  onSecondaryAction,
  className,
}) => {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center p-8 sm:p-12 text-center rounded-lg bg-card border border-border shadow-sm",
        className
      )}
    >
      {/* Wireframe / Soft Icon Tile matching BMS reference */}
      <div className="relative mb-5 flex items-center justify-center">
        <div className="w-16 h-16 rounded-2xl bg-ember-50 dark:bg-ember-950/40 border border-ember-100 dark:border-ember-900/60 flex items-center justify-center shadow-inner">
          <Icon className="w-8 h-8 text-ember-600 dark:text-ember-400" />
        </div>
      </div>

      <h3 className="text-lg font-semibold text-foreground tracking-tight mb-1.5">{title}</h3>
      <p className="text-sm text-muted-foreground max-w-sm mb-6 leading-relaxed">{description}</p>

      {(actionLabel || secondaryActionLabel) && (
        <div className="flex flex-wrap items-center justify-center gap-3">
          {actionLabel && (
            <button
              type="button"
              onClick={onAction}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-ember-500 hover:bg-ember-600 text-ember-950 font-semibold text-sm shadow-sm transition-all active:scale-[0.98]"
            >
              <Plus className="w-4 h-4" />
              <span>{actionLabel}</span>
            </button>
          )}

          {secondaryActionLabel && (
            <button
              type="button"
              onClick={onSecondaryAction}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-card hover:bg-muted text-foreground border border-border font-medium text-sm transition-colors"
            >
              <span>{secondaryActionLabel}</span>
            </button>
          )}
        </div>
      )}
    </div>
  );
};
