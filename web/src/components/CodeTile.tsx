import { Check, Clock, Copy, ExternalLink, KeyRound } from "lucide-react";
import type React from "react";
import { useState } from "react";
import { cn } from "../lib/utils";

interface CodeTileProps {
  code: string;
  source?: "Verify" | "Found in message";
  size?: "compact" | "large" | "grid";
  isNew?: boolean;
  expiresInSeconds?: number;
  className?: string;
  linkUrl?: string;
}

export const CodeTile: React.FC<CodeTileProps> = ({
  code,
  source = "Found in message",
  size = "compact",
  isNew = false,
  expiresInSeconds,
  className,
  linkUrl,
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch {
      // Fallback
    }
  };

  // Format code by grouping digits into chunks of 3 if numeric
  const isNumeric = /^\d+$/.test(code);
  const formattedCode = isNumeric ? code.replace(/\B(?=(\d{3})+(?!\d))/g, " ") : code;

  if (size === "compact") {
    return (
      <div
        className={cn(
          "inline-flex items-center gap-2 px-2.5 py-1 rounded-md bg-ember-50 dark:bg-ember-950/30 border border-ember-200 dark:border-ember-800/60 text-ember-950 dark:text-ember-100 transition-all",
          isNew && "animate-pulse-once",
          className
        )}
      >
        <KeyRound
          className="w-3.5 h-3.5 text-ember-600 dark:text-ember-400 shrink-0"
          aria-hidden="true"
        />
        <span
          className="font-mono text-xs font-bold tracking-wider tabular-nums select-all"
          aria-label={code}
        >
          {formattedCode}
        </span>
        <button
          type="button"
          onClick={handleCopy}
          className="p-1 -mr-1 rounded hover:bg-ember-100 dark:hover:bg-ember-900/50 text-ember-700 dark:text-ember-300 transition-colors"
          title={`Copy code ${code}`}
          aria-label={`Copy code ${code}`}
        >
          {copied ? (
            <Check className="w-3.5 h-3.5 text-emerald-600" />
          ) : (
            <Copy className="w-3.5 h-3.5" />
          )}
        </button>
      </div>
    );
  }

  // Large or Grid tile
  return (
    <div
      className={cn(
        "relative flex flex-col justify-between p-4 rounded-lg bg-ember-50 dark:bg-ember-950/30 border border-ember-200 dark:border-ember-800/70 shadow-sm transition-all",
        isNew && "animate-pulse-once",
        className
      )}
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-md bg-ember-100 dark:bg-ember-900/50 flex items-center justify-center text-ember-700 dark:text-ember-300">
            <KeyRound className="w-4 h-4" />
          </div>
          <span className="text-xs font-medium text-ember-800 dark:text-ember-300">{source}</span>
        </div>

        {expiresInSeconds !== undefined && expiresInSeconds > 0 && (
          <span className="inline-flex items-center gap-1 text-xs text-neutral-500 tabular-nums">
            <Clock className="w-3 h-3" />
            {Math.floor(expiresInSeconds / 60)}:{String(expiresInSeconds % 60).padStart(2, "0")}
          </span>
        )}
      </div>

      <div className="my-2">
        <div
          className="font-mono text-2xl lg:text-3xl font-bold tracking-wider text-ember-950 dark:text-ember-50 tabular-nums select-all"
          aria-label={code}
        >
          {formattedCode}
        </div>
      </div>

      <div className="flex items-center justify-between pt-2 border-t border-ember-200/60 dark:border-ember-800/50">
        <span className="text-xs text-neutral-500 dark:text-neutral-400">
          {linkUrl ? "One-time link" : "Verification code"}
        </span>
        <div className="flex items-center gap-1.5">
          {linkUrl && (
            <a
              href={linkUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-xs font-medium px-2.5 py-1.5 rounded-md bg-white dark:bg-neutral-800 text-neutral-700 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-50 transition-colors"
            >
              <ExternalLink className="w-3.5 h-3.5" />
              Open
            </a>
          )}
          <button
            type="button"
            onClick={handleCopy}
            className={cn(
              "inline-flex items-center gap-1.5 text-xs font-medium px-3 py-1.5 rounded-md transition-colors",
              copied
                ? "bg-emerald-600 text-white"
                : "bg-ember-500 hover:bg-ember-600 text-ember-950 font-semibold"
            )}
            aria-label={`Copy code ${code}`}
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5" />
                <span>Copied</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                <span>Copy</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
