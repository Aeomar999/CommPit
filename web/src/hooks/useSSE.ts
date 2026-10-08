import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

export function useSSE(project = "default") {
  const queryClient = useQueryClient();
  const [isConnected, setIsConnected] = useState(false);
  const [lastEventTime, setLastEventTime] = useState<Date | null>(null);

  useEffect(() => {
    let eventSource: EventSource | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    function connect() {
      try {
        const url = `/api/v1/events?project=${encodeURIComponent(project)}`;
        eventSource = new EventSource(url, { withCredentials: true });

        eventSource.onopen = () => {
          setIsConnected(true);
        };

        const handleUpdate = () => {
          setLastEventTime(new Date());
          queryClient.invalidateQueries({ queryKey: ["messages"] });
          queryClient.invalidateQueries({ queryKey: ["verifications"] });
          queryClient.invalidateQueries({ queryKey: ["batches"] });
        };

        const handleRequestLogged = () => {
          setLastEventTime(new Date());
          queryClient.invalidateQueries({ queryKey: ["request-logs"] });
        };

        eventSource.addEventListener("message.created", handleUpdate);
        eventSource.addEventListener("message.updated", handleUpdate);
        eventSource.addEventListener("message.status", handleUpdate);
        eventSource.addEventListener("verification.created", handleUpdate);
        eventSource.addEventListener("verification.updated", handleUpdate);
        eventSource.addEventListener("batch.updated", handleUpdate);
        eventSource.addEventListener("request.logged", handleRequestLogged);

        eventSource.onerror = () => {
          setIsConnected(false);
          eventSource?.close();
          retryTimer = setTimeout(connect, 3000);
        };
      } catch {
        setIsConnected(false);
        retryTimer = setTimeout(connect, 3000);
      }
    }

    connect();

    return () => {
      if (retryTimer) clearTimeout(retryTimer);
      if (eventSource) eventSource.close();
    };
  }, [project, queryClient]);

  return { isConnected, lastEventTime };
}
