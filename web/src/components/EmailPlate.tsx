import { Image, ShieldCheck } from "lucide-react";
import React, { useState } from "react";
import { cn } from "../lib/utils";
import type { Message } from "../types";

interface EmailPlateProps {
  message: Message;
  className?: string;
}

export const EmailPlate: React.FC<EmailPlateProps> = ({ message, className }) => {
  const [activeTab, setActiveTab] = useState<"html" | "text" | "headers" | "source">("html");
  const [allowImages, setAllowImages] = useState(false);

  // Sanitized iframe HTML representation
  const iframeContent = React.useMemo(() => {
    let html = message.body_html || "";
    if (!allowImages) {
      // Strip src attributes from img tags or block them
      html = html.replace(
        /<img\s+([^>]*?)src=["'](.*?)["']/gi,
        '<img $1data-blocked-src="$2" alt="[Image blocked for security]"'
      );
    }
    return html;
  }, [message.body_html, allowImages]);

  return (
    <div
      className={cn("rounded-lg border border-border bg-card overflow-hidden shadow-sm", className)}
    >
      {/* Plate Header with Tabs */}
      <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 bg-muted/40 border-b border-border">
        <div className="flex items-center gap-1 p-1 bg-muted rounded-md text-xs font-medium">
          <button
            type="button"
            onClick={() => setActiveTab("html")}
            className={cn(
              "px-3 py-1.5 rounded-sm transition-all",
              activeTab === "html"
                ? "bg-card text-foreground shadow-sm font-semibold"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            HTML Preview
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("text")}
            className={cn(
              "px-3 py-1.5 rounded-sm transition-all",
              activeTab === "text"
                ? "bg-card text-foreground shadow-sm font-semibold"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            Plain Text
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("source")}
            className={cn(
              "px-3 py-1.5 rounded-sm transition-all",
              activeTab === "source"
                ? "bg-card text-foreground shadow-sm font-semibold"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            Source
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("headers")}
            className={cn(
              "px-3 py-1.5 rounded-sm transition-all",
              activeTab === "headers"
                ? "bg-card text-foreground shadow-sm font-semibold"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            Headers
          </button>
        </div>

        {activeTab === "html" && message.body_html && (
          <button
            type="button"
            onClick={() => setAllowImages(!allowImages)}
            className={cn(
              "inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium transition-colors border",
              allowImages
                ? "bg-ember-50 border-ember-300 text-ember-800 dark:bg-ember-950/40 dark:text-ember-300"
                : "bg-card border-border text-muted-foreground hover:bg-muted"
            )}
          >
            <Image className="w-3.5 h-3.5" />
            <span>{allowImages ? "Images Enabled" : "Load Remote Images"}</span>
          </button>
        )}
      </div>

      {/* Security sandbox notice */}
      <div className="flex items-center gap-2 px-4 py-2 bg-neutral-50 dark:bg-neutral-900/40 border-b border-border text-[11px] text-muted-foreground">
        <ShieldCheck className="w-3.5 h-3.5 text-emerald-600 shrink-0" />
        <span>
          Rendered in a sandbox iframe. Scripts are disabled and local storage is isolated.
        </span>
      </div>

      {/* Plate Body */}
      <div className="p-4 bg-background">
        {activeTab === "html" &&
          (message.body_html ? (
            <div className="w-full min-h-[360px] rounded-md border border-neutral-200 dark:border-neutral-800 bg-white overflow-hidden">
              <iframe
                title="Email HTML Preview"
                srcDoc={`<!DOCTYPE html><html><head><meta charset="utf-8"><style>body{font-family:system-ui,sans-serif;margin:16px;color:#181D24;background:#fff;}</style></head><body>${iframeContent}</body></html>`}
                sandbox=""
                className="w-full min-h-[360px] border-0"
              />
            </div>
          ) : (
            <div className="py-12 text-center text-sm text-muted-foreground">
              No HTML content was provided for this email.
            </div>
          ))}

        {activeTab === "text" && (
          <div className="p-4 rounded-md bg-muted/40 border border-border font-mono text-xs whitespace-pre-wrap leading-relaxed text-foreground">
            {message.body_text || message.body_html || "No plain text content."}
          </div>
        )}

        {activeTab === "source" && (
          <div className="p-4 rounded-md bg-muted/40 border border-border font-mono text-xs whitespace-pre-wrap leading-relaxed text-foreground overflow-x-auto">
            {message.body_html || message.body_text || "No raw source content."}
          </div>
        )}

        {activeTab === "headers" && (
          <div className="space-y-2 font-mono text-xs">
            <div className="p-3 rounded-md bg-muted/40 border border-border space-y-1">
              <div>
                <span className="text-muted-foreground">From:</span> {message.from}
              </div>
              <div>
                <span className="text-muted-foreground">To:</span> {message.to}
              </div>
              {message.subject && (
                <div>
                  <span className="text-muted-foreground">Subject:</span> {message.subject}
                </div>
              )}
              <div>
                <span className="text-muted-foreground">Message-ID:</span> {message.id}
              </div>
              <div>
                <span className="text-muted-foreground">Provider:</span> {message.provider}
              </div>
              <div>
                <span className="text-muted-foreground">Date:</span>{" "}
                {new Date(message.created_at).toUTCString()}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
