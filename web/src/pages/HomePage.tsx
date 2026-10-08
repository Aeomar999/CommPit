import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Code2, CornerDownLeft, Mail, MessageSquare, RefreshCw, Send, Trash2 } from "lucide-react";
import { useContext, useMemo, useState } from "react";
import { CodeTile } from "../components/CodeTile";
import { DeliveryTrack } from "../components/DeliveryTrack";
import { EmailPlate } from "../components/EmailPlate";
import { EmptyState } from "../components/EmptyState";
import { LayoutContext } from "../components/Layout";
import { OnboardingChecklist } from "../components/OnboardingChecklist";
import { StatusChip } from "../components/StatusChip";
import { messagesApi } from "../lib/api";
import { cn } from "../lib/utils";

export function HomePage() {
  const queryClient = useQueryClient();
  const { searchQuery, openCompose } = useContext(LayoutContext);

  const [activeChannel, setActiveChannel] = useState<"all" | "sms" | "email" | "inbound">("all");
  const [selectedMessageId, setSelectedMessageId] = useState<string | null>(null);
  const [showRawJson, setShowRawJson] = useState(false);
  const [showOnboarding, setShowOnboarding] = useState(true);

  // Fetch all messages
  const {
    data: messagesResponse,
    isLoading,
    refetch,
    isRefetching,
  } = useQuery({
    queryKey: ["messages"],
    queryFn: () => messagesApi.list({ limit: 100 }),
  });

  // Clear messages mutation
  const clearMutation = useMutation({
    mutationFn: () => messagesApi.delete("default"),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["messages"] });
      setSelectedMessageId(null);
    },
  });

  const messages = messagesResponse?.messages || [];

  // Filter messages based on channel tab and global search query
  const filteredMessages = useMemo(() => {
    return messages.filter((msg) => {
      // Channel filtering
      if (activeChannel === "sms" && msg.channel !== "sms") return false;
      if (activeChannel === "email" && msg.channel !== "email") return false;
      if (activeChannel === "inbound" && msg.direction !== "inbound") return false;

      // Search query filtering
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase();
        const matchesTo = msg.to?.toLowerCase().includes(query);
        const matchesFrom = msg.from?.toLowerCase().includes(query);
        const matchesBody =
          msg.body_text?.toLowerCase().includes(query) ||
          msg.body_html?.toLowerCase().includes(query);
        const matchesSubject = msg.subject?.toLowerCase().includes(query);
        const matchesId = msg.id.toLowerCase().includes(query);
        const matchesCodes = msg.extracted_codes?.some((c) => c.toLowerCase().includes(query));

        if (
          !matchesTo &&
          !matchesFrom &&
          !matchesBody &&
          !matchesSubject &&
          !matchesId &&
          !matchesCodes
        ) {
          return false;
        }
      }

      return true;
    });
  }, [messages, activeChannel, searchQuery]);

  // Selected message
  const selectedMessage = useMemo(() => {
    if (!messages.length) return null;
    if (selectedMessageId) {
      return messages.find((m) => m.id === selectedMessageId) || messages[0];
    }
    return filteredMessages[0] || messages[0];
  }, [messages, selectedMessageId, filteredMessages]);

  const handleClearInbox = () => {
    if (window.confirm("Are you sure you want to clear all messages in this project?")) {
      clearMutation.mutate();
    }
  };

  return (
    <div className="space-y-6">
      {/* Top Action & Segmented Filter Bar inspired by BMS reference shots */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        {/* Segmented Channel Tabs */}
        <div className="flex items-center gap-1 p-1 bg-muted rounded-lg text-xs font-medium self-start">
          <button
            type="button"
            onClick={() => setActiveChannel("all")}
            className={cn(
              "px-3 py-1.5 rounded-md transition-all",
              activeChannel === "all"
                ? "bg-card text-foreground shadow-sm font-semibold"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            All Messages ({messages.length})
          </button>
          <button
            type="button"
            onClick={() => setActiveChannel("sms")}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 rounded-md transition-all",
              activeChannel === "sms"
                ? "bg-card text-foreground shadow-sm font-semibold"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            <MessageSquare className="w-3.5 h-3.5 text-ember-600" />
            <span>SMS</span>
          </button>
          <button
            type="button"
            onClick={() => setActiveChannel("email")}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 rounded-md transition-all",
              activeChannel === "email"
                ? "bg-card text-foreground shadow-sm font-semibold"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            <Mail className="w-3.5 h-3.5 text-blue-600" />
            <span>Email</span>
          </button>
          <button
            type="button"
            onClick={() => setActiveChannel("inbound")}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 rounded-md transition-all",
              activeChannel === "inbound"
                ? "bg-card text-foreground shadow-sm font-semibold"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            <CornerDownLeft className="w-3.5 h-3.5 text-teal-600" />
            <span>Inbound</span>
          </button>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2 self-end sm:self-auto">
          <button
            type="button"
            onClick={() => refetch()}
            disabled={isRefetching}
            className="p-2 rounded-md border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            title="Refresh inbox"
            aria-label="Refresh inbox"
          >
            <RefreshCw className={cn("w-4 h-4", isRefetching && "animate-spin")} />
          </button>

          {messages.length > 0 && (
            <button
              type="button"
              onClick={handleClearInbox}
              className="inline-flex items-center gap-1.5 px-3 py-2 rounded-md border border-border bg-card hover:bg-red-50 dark:hover:bg-red-950/30 text-xs font-medium text-muted-foreground hover:text-red-700 dark:hover:text-red-400 transition-colors"
            >
              <Trash2 className="w-3.5 h-3.5" />
              <span>Clear</span>
            </button>
          )}

          <button
            type="button"
            onClick={openCompose}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-md bg-ember-500 hover:bg-ember-600 text-ember-950 font-semibold text-xs shadow-sm transition-all active:scale-[0.98]"
          >
            <Send className="w-3.5 h-3.5" />
            <span>Compose</span>
          </button>
        </div>
      </div>

      {/* When Empty: Show BMS-style empty card and onboarding checklist */}
      {messages.length === 0 && !isLoading && (
        <div className="space-y-6">
          <EmptyState
            title="No Messages in Sandbox"
            description="Your mocksms inbox is empty. Send an SMS via the REST API, dispatch an email via SMTP (port 1025), or use the test composer."
            actionLabel="Send Test Message"
            onAction={openCompose}
            secondaryActionLabel="Refresh Feed"
            onSecondaryAction={() => refetch()}
          />

          {showOnboarding && <OnboardingChecklist onDismiss={() => setShowOnboarding(false)} />}
        </div>
      )}

      {/* When Messages Exist: Split Master-Detail Layout */}
      {messages.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
          {/* Master Message List (Left 5 Cols) */}
          <div className="lg:col-span-5 rounded-xl bg-card border border-border shadow-xs overflow-hidden">
            <div className="p-3 bg-muted/30 border-b border-border flex items-center justify-between text-xs text-muted-foreground">
              <span className="font-semibold text-foreground">
                Messages ({filteredMessages.length})
              </span>
              <span>Sorted newest first</span>
            </div>

            <div className="divide-y divide-border max-h-[750px] overflow-y-auto">
              {filteredMessages.length === 0 ? (
                <div className="p-8 text-center text-xs text-muted-foreground">
                  No messages matching current filter or search.
                </div>
              ) : (
                filteredMessages.map((msg) => {
                  const isSelected = selectedMessage?.id === msg.id;
                  const hasCode = msg.extracted_codes && msg.extracted_codes.length > 0;
                  const isSms = msg.channel === "sms";

                  return (
                    <button
                      type="button"
                      key={msg.id}
                      onClick={() => setSelectedMessageId(msg.id)}
                      className={cn(
                        "w-full p-4 cursor-pointer transition-all text-left group block",
                        isSelected
                          ? "bg-ember-50/70 dark:bg-ember-950/30 border-l-4 border-l-ember-500 pl-3"
                          : "hover:bg-muted/40"
                      )}
                    >
                      <div className="flex items-start justify-between gap-2 mb-1.5">
                        <div className="flex items-center gap-2 overflow-hidden">
                          {/* Channel Icon Tile */}
                          <div
                            className={cn(
                              "w-7 h-7 rounded-lg flex items-center justify-center shrink-0 text-xs",
                              isSms
                                ? "bg-ember-100 text-ember-700 dark:bg-ember-950 dark:text-ember-300"
                                : "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300"
                            )}
                          >
                            {isSms ? (
                              <MessageSquare className="w-3.5 h-3.5" />
                            ) : (
                              <Mail className="w-3.5 h-3.5" />
                            )}
                          </div>

                          <span className="text-xs font-bold text-foreground truncate font-mono">
                            {msg.to}
                          </span>
                        </div>

                        <StatusChip status={msg.status} isSimulated={true} />
                      </div>

                      {/* Snippet / Content Preview */}
                      <p className="text-xs text-muted-foreground line-clamp-2 leading-relaxed mb-2">
                        {msg.subject ? (
                          <span className="font-semibold text-foreground mr-1">
                            {msg.subject} -
                          </span>
                        ) : null}
                        {msg.body_text ||
                          msg.body_html?.replace(/<[^>]+>/g, "") ||
                          "(Empty message body)"}
                      </p>

                      {/* Bottom row: compact code tile & metadata */}
                      <div className="flex items-center justify-between gap-2 pt-1 border-t border-border/50 text-[11px] text-muted-foreground">
                        <div className="flex items-center gap-2">
                          <span className="px-1.5 py-0.5 rounded bg-muted text-[10px] uppercase font-semibold font-mono">
                            {msg.provider}
                          </span>
                          <span className="font-mono tabular-nums">
                            {new Date(msg.created_at).toLocaleTimeString([], {
                              hour: "2-digit",
                              minute: "2-digit",
                            })}
                          </span>
                        </div>

                        {/* Compact Signature Code Tile */}
                        {hasCode && (
                          <CodeTile
                            code={msg.extracted_codes![0]}
                            size="compact"
                            source="Found in message"
                          />
                        )}
                      </div>
                    </button>
                  );
                })
              )}
            </div>
          </div>

          {/* Detail Pane (Right 7 Cols) */}
          <div className="lg:col-span-7 space-y-4">
            {selectedMessage ? (
              <div className="rounded-xl bg-card border border-border shadow-xs overflow-hidden">
                {/* Header */}
                <div className="p-6 border-b border-border bg-card">
                  <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
                    <div>
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                          {selectedMessage.channel.toUpperCase()} Message
                        </span>
                        <span className="text-muted-foreground">•</span>
                        <span className="text-xs font-mono text-muted-foreground">
                          ID: {selectedMessage.id}
                        </span>
                      </div>
                      <h2 className="text-xl font-bold text-foreground font-mono">
                        {selectedMessage.to}
                      </h2>
                    </div>

                    <div className="flex items-center gap-2">
                      <StatusChip status={selectedMessage.status} isSimulated={true} />
                      <button
                        type="button"
                        onClick={() => setShowRawJson(!showRawJson)}
                        className={cn(
                          "p-1.5 rounded-md border text-xs font-mono transition-colors",
                          showRawJson
                            ? "bg-neutral-900 text-white border-neutral-900"
                            : "bg-card border-border text-muted-foreground hover:bg-muted"
                        )}
                        title="Toggle raw JSON payload"
                      >
                        <Code2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  {/* Metadata Chips */}
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 p-3 rounded-lg bg-muted/40 border border-border text-xs">
                    <div>
                      <span className="text-muted-foreground block text-[11px]">From / Sender</span>
                      <span className="font-semibold text-foreground font-mono truncate block">
                        {selectedMessage.from || "MOCKSMS"}
                      </span>
                    </div>
                    <div>
                      <span className="text-muted-foreground block text-[11px]">
                        Provider Adapter
                      </span>
                      <span className="font-semibold text-foreground uppercase font-mono">
                        {selectedMessage.provider}
                      </span>
                    </div>
                    <div>
                      <span className="text-muted-foreground block text-[11px]">
                        Segments / Chars
                      </span>
                      <span className="font-semibold text-foreground font-mono">
                        {selectedMessage.segments || 1} seg (
                        {selectedMessage.body_text?.length || 0} chars)
                      </span>
                    </div>
                    <div>
                      <span className="text-muted-foreground block text-[11px]">Received At</span>
                      <span className="font-semibold text-foreground font-mono tabular-nums">
                        {new Date(selectedMessage.created_at).toLocaleTimeString()}
                      </span>
                    </div>
                  </div>
                </div>

                <div className="p-6 space-y-6">
                  {/* Large Signature Code Tile if OTP/link found */}
                  {selectedMessage.extracted_codes &&
                    selectedMessage.extracted_codes.length > 0 && (
                      <CodeTile
                        code={selectedMessage.extracted_codes[0]}
                        size="large"
                        source="Found in message"
                        linkUrl={selectedMessage.primary_link || undefined}
                      />
                    )}

                  {/* Delivery Timeline Track */}
                  <DeliveryTrack
                    status={selectedMessage.status}
                    createdAt={selectedMessage.created_at}
                    errorCode={selectedMessage.error_code || undefined}
                    errorMessage={selectedMessage.error_message || undefined}
                  />

                  {/* Raw JSON View if toggled */}
                  {showRawJson ? (
                    <div className="p-4 rounded-lg bg-neutral-950 text-neutral-100 font-mono text-xs overflow-x-auto select-all leading-relaxed">
                      <pre>{JSON.stringify(selectedMessage, null, 2)}</pre>
                    </div>
                  ) : (
                    <>
                      {/* SMS Chat Bubble or Email Plate */}
                      {selectedMessage.channel === "sms" ? (
                        <div className="p-5 rounded-xl bg-neutral-50 dark:bg-neutral-900/40 border border-border">
                          <div className="text-xs font-semibold text-muted-foreground mb-3 uppercase tracking-wider">
                            Simulated SMS Conversation
                          </div>

                          <div className="flex flex-col space-y-3">
                            <div
                              className={cn(
                                "max-w-[85%] p-4 rounded-2xl shadow-xs leading-relaxed text-sm",
                                selectedMessage.direction === "inbound"
                                  ? "self-start bg-teal-50 dark:bg-teal-950/40 border border-teal-200 dark:border-teal-800 text-teal-950 dark:text-teal-100 rounded-bl-none"
                                  : "self-end bg-card border border-border text-foreground rounded-br-none"
                              )}
                            >
                              <div className="whitespace-pre-wrap font-sans">
                                {selectedMessage.body_text}
                              </div>
                              <div className="mt-2 flex items-center justify-end gap-1 text-[11px] text-muted-foreground tabular-nums font-mono">
                                <span>
                                  {new Date(selectedMessage.created_at).toLocaleTimeString([], {
                                    hour: "2-digit",
                                    minute: "2-digit",
                                  })}
                                </span>
                              </div>
                            </div>
                          </div>
                        </div>
                      ) : (
                        <EmailPlate message={selectedMessage} />
                      )}
                    </>
                  )}
                </div>
              </div>
            ) : (
              <div className="p-12 text-center rounded-xl bg-card border border-border text-muted-foreground text-sm">
                Select a message to view full details
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
