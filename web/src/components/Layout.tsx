import { useQuery } from "@tanstack/react-query";
import React, { useState } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { useSSE } from "../hooks/useSSE";
import { messagesApi, verificationsApi } from "../lib/api";
import { ComposeModal } from "./ComposeModal";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";

export const LayoutContext = React.createContext<{
  searchQuery: string;
  setSearchQuery: (q: string) => void;
  openCompose: () => void;
}>({
  searchQuery: "",
  setSearchQuery: () => {},
  openCompose: () => {},
});

export function Layout() {
  const [collapsed, setCollapsed] = useState(false);
  const [composeOpen, setComposeOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const location = useLocation();

  // Connect to live SSE stream for real-time updates
  const { isConnected } = useSSE("default");

  // Fetch messages and verifications counts for header stats
  const { data: messagesData, refetch: refetchMessages } = useQuery({
    queryKey: ["messages"],
    queryFn: () => messagesApi.list({ limit: 100 }),
  });

  const { data: verificationsData } = useQuery({
    queryKey: ["verifications"],
    queryFn: () => verificationsApi.list({ limit: 50 }),
  });

  const messages = messagesData?.messages || [];
  const totalCount = messages.length;
  const deliveredCount = messages.filter((m) => m.status === "delivered").length;
  const failedCount = messages.filter(
    (m) => m.status === "failed" || m.status === "undelivered"
  ).length;
  const otpCount = verificationsData?.verifications?.length || 0;

  // Determine current page title
  const getPageTitle = () => {
    switch (location.pathname) {
      case "/inbox":
        return "Inbox";
      case "/otps":
        return "OTPs & Verifications";
      case "/batches":
        return "Batches";
      case "/inspector":
        return "Inspector";
      case "/settings":
        return "Settings";
      default:
        return "mocksms";
    }
  };

  return (
    <LayoutContext.Provider
      value={{
        searchQuery,
        setSearchQuery,
        openCompose: () => setComposeOpen(true),
      }}
    >
      <div className="min-h-screen bg-background flex">
        {/* Left Sidebar */}
        <Sidebar
          collapsed={collapsed}
          onToggleCollapse={() => setCollapsed(!collapsed)}
          unreadCount={messages.length}
          otpCount={otpCount}
        />

        {/* Main Content Area */}
        <div className="flex-1 flex flex-col min-w-0">
          <TopBar
            title={getPageTitle()}
            totalCount={totalCount}
            deliveredCount={deliveredCount}
            failedCount={failedCount}
            onOpenCompose={() => setComposeOpen(true)}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            isConnected={isConnected}
          />

          <main className="flex-1 p-4 sm:p-6 lg:p-8 max-w-7xl w-full mx-auto">
            <Outlet />
          </main>
        </div>

        {/* Test message compose modal */}
        <ComposeModal
          isOpen={composeOpen}
          onClose={() => setComposeOpen(false)}
          onSuccess={() => refetchMessages()}
        />
      </div>
    </LayoutContext.Provider>
  );
}
