import { ChevronDown, Layers, Plus, Search } from "lucide-react";
import { SandboxPill } from "./SandboxPill";

interface TopBarProps {
  title: string;
  totalCount: number;
  deliveredCount: number;
  failedCount: number;
  onOpenCompose: () => void;
  searchQuery: string;
  onSearchChange: (q: string) => void;
  isConnected: boolean;
  project?: string;
}

export const TopBar: React.FC<TopBarProps> = ({
  title,
  totalCount,
  deliveredCount,
  failedCount,
  onOpenCompose,
  searchQuery,
  onSearchChange,
  isConnected,
  project = "default",
}) => {
  return (
    <header className="sticky top-0 z-20 flex items-center justify-between h-16 px-6 bg-card/95 backdrop-blur-md border-b border-border shadow-xs">
      {/* Left: Page Title */}
      <div className="flex items-center gap-4">
        <h1 className="text-xl font-bold tracking-tight text-foreground">{title}</h1>

        {/* Live SSE pulse indicator */}
        <div
          className="hidden sm:flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-neutral-100 dark:bg-neutral-800 text-muted-foreground"
          title={isConnected ? "Live real-time feed connected" : "Reconnecting to event stream..."}
        >
          <span
            className={`w-2 h-2 rounded-full ${
              isConnected ? "bg-emerald-500 animate-pulse" : "bg-amber-500 animate-ping"
            }`}
          />
          <span className="tabular-nums">{isConnected ? "Live" : "Syncing"}</span>
        </div>
      </div>

      {/* Center: Search input */}
      <div className="hidden md:flex items-center flex-1 max-w-xs mx-6">
        <div className="relative w-full">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="search"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search messages, OTPs, numbers... (Ctrl+K)"
            className="w-full pl-9 pr-3 py-1.5 rounded-md border border-border bg-background text-foreground text-xs placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ember-500"
          />
        </div>
      </div>

      {/* Right side items inspired by BMS stats and sandbox */}
      <div className="flex items-center gap-3">
        {/* BMS-inspired stats pill card */}
        <div className="hidden lg:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-neutral-50 dark:bg-neutral-900 border border-neutral-200/80 dark:border-neutral-800 text-xs font-mono">
          <span className="text-muted-foreground">Total:</span>
          <span className="font-semibold text-foreground tabular-nums">{totalCount}</span>
          <span className="text-neutral-300 dark:text-neutral-700">|</span>
          <span className="text-emerald-700 dark:text-emerald-400">Delivered:</span>
          <span className="font-semibold text-emerald-700 dark:text-emerald-400 tabular-nums">
            {deliveredCount}
          </span>
          {failedCount > 0 && (
            <>
              <span className="text-neutral-300 dark:text-neutral-700">|</span>
              <span className="text-red-600 dark:text-red-400">Failed:</span>
              <span className="font-semibold text-red-600 dark:text-red-400 tabular-nums">
                {failedCount}
              </span>
            </>
          )}
        </div>

        {/* Project Selector Badge */}
        <div className="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted text-xs font-medium text-foreground border border-border">
          <Layers className="w-3.5 h-3.5 text-muted-foreground" />
          <span>{project}</span>
          <ChevronDown className="w-3 h-3 text-muted-foreground" />
        </div>

        {/* The Sandbox Pill (crucial per spec §10.12) */}
        <SandboxPill />

        {/* Primary Action Button */}
        <button
          type="button"
          onClick={onOpenCompose}
          className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-md bg-ember-500 hover:bg-ember-600 text-ember-950 font-semibold text-xs shadow-sm transition-all active:scale-[0.98]"
        >
          <Plus className="w-4 h-4" />
          <span className="hidden sm:inline">Send Test</span>
        </button>
      </div>
    </header>
  );
};
