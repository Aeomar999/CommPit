import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Send } from "lucide-react";
import { useState } from "react";
import { messagesApi } from "../lib/api";
import { cn } from "../lib/utils";

export function ReplyBox({ from, to }: { from: string; to: string }) {
  const queryClient = useQueryClient();
  const [body, setBody] = useState("");
  const [error, setError] = useState<string | null>(null);

  const reply = useMutation({
    mutationFn: () => messagesApi.simulateInbound({ from, to, body: body.trim() }),
    onSuccess: () => {
      setBody("");
      setError(null);
      queryClient.invalidateQueries({ queryKey: ["messages"] });
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : "Reply failed. Check the server and try again.");
    },
  });

  return (
    <div className="rounded-xl bg-card border border-border shadow-xs p-5 space-y-3">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        Reply as {from}
      </h3>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (body.trim() && !reply.isPending) {
            reply.mutate();
          }
        }}
        className="space-y-3"
      >
        <div className="space-y-1.5">
          <label htmlFor="reply-body" className="sr-only">
            Reply text
          </label>
          <textarea
            id="reply-body"
            rows={2}
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder="Type a reply from the handset…"
            disabled={reply.isPending}
            className="w-full p-3 rounded-md border border-border bg-card text-foreground text-sm focus:outline-none focus:ring-2 focus:ring-ember-500 leading-relaxed placeholder:text-muted-foreground disabled:opacity-50"
          />
        </div>
        {error && (
          <p role="alert" className="text-xs text-red-600 dark:text-red-400 font-medium">
            {error}
          </p>
        )}
        <div className="flex flex-wrap items-center gap-3">
          <p className="text-[11px] text-muted-foreground">
            Tip: reply STOP to test opt-out, START to resubscribe.
          </p>
          <button
            type="submit"
            disabled={!body.trim() || reply.isPending}
            className={cn(
              "ml-auto inline-flex items-center gap-2 px-4 py-2 rounded-md font-semibold text-xs shadow-sm transition-all active:scale-[0.98]",
              "bg-ember-500 hover:bg-ember-600 text-ember-950 disabled:opacity-50"
            )}
          >
            <Send className="w-3.5 h-3.5" />
            <span>{reply.isPending ? "Sending…" : "Send reply"}</span>
          </button>
        </div>
      </form>
    </div>
  );
}
