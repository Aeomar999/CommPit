import {
  ChevronsLeft,
  ChevronsRight,
  Inbox,
  KeyRound,
  Layers,
  MessageSquare,
  Moon,
  Server,
  Settings,
  Sun,
  Terminal,
} from "lucide-react";
import React from "react";
import { NavLink } from "react-router-dom";
import { cn } from "../lib/utils";

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  unreadCount?: number;
  otpCount?: number;
}

export const Sidebar: React.FC<SidebarProps> = ({
  collapsed,
  onToggleCollapse,
  unreadCount = 0,
  otpCount = 0,
}) => {
  const [isDark, setIsDark] = React.useState(false);

  const toggleTheme = () => {
    setIsDark(!isDark);
    if (!isDark) {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
  };

  const navItems = [
    {
      to: "/inbox",
      label: "Inbox",
      icon: Inbox,
      badge: unreadCount > 0 ? unreadCount : undefined,
    },
    {
      to: "/otps",
      label: "OTPs & Codes",
      icon: KeyRound,
      badge: otpCount > 0 ? otpCount : undefined,
    },
    {
      to: "/batches",
      label: "Batches",
      icon: Layers,
    },
    {
      to: "/inspector",
      label: "Inspector",
      icon: Terminal,
    },
    {
      to: "/settings",
      label: "Settings",
      icon: Settings,
    },
  ];

  return (
    <aside
      className={cn(
        "sticky top-0 h-screen z-30 flex flex-col justify-between bg-card border-r border-border transition-all duration-300 ease-in-out shrink-0 select-none",
        collapsed ? "w-[72px]" : "w-64"
      )}
    >
      {/* Top Header / Brand Logo */}
      <div>
        <div className="flex items-center justify-between h-16 px-4 border-b border-border">
          <div className="flex items-center gap-3 overflow-hidden">
            {/* Dashed orange bubble icon inspired by BMS */}
            <div className="w-9 h-9 rounded-xl bg-ember-500/10 border border-dashed border-ember-600 flex items-center justify-center shrink-0 shadow-inner">
              <MessageSquare className="w-5 h-5 text-ember-600 dark:text-ember-500" />
            </div>

            {!collapsed && (
              <div className="flex items-baseline">
                <span className="text-lg font-extrabold tracking-tight text-foreground font-sans">
                  mock
                </span>
                <span className="text-lg font-extrabold tracking-tight text-ember-600 dark:text-ember-400 font-sans">
                  sms
                </span>
              </div>
            )}
          </div>

          <button
            type="button"
            onClick={onToggleCollapse}
            className="p-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          >
            {collapsed ? (
              <ChevronsRight className="w-4 h-4" />
            ) : (
              <ChevronsLeft className="w-4 h-4" />
            )}
          </button>
        </div>

        {/* Navigation items */}
        <nav className="p-3 space-y-1.5">
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  cn(
                    "relative flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all group",
                    isActive
                      ? "bg-ember-50 dark:bg-ember-950/40 text-ember-700 dark:text-ember-300 font-semibold"
                      : "text-muted-foreground hover:bg-muted hover:text-foreground"
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    {/* Active accent left bar */}
                    {isActive && (
                      <span className="absolute left-0 top-1.5 bottom-1.5 w-1 rounded-r-full bg-ember-500" />
                    )}

                    <Icon
                      className={cn(
                        "w-5 h-5 shrink-0 transition-colors",
                        isActive
                          ? "text-ember-600 dark:text-ember-400"
                          : "text-muted-foreground group-hover:text-foreground"
                      )}
                    />

                    {!collapsed && <span className="flex-1 truncate">{item.label}</span>}

                    {!collapsed && item.badge !== undefined && (
                      <span className="px-2 py-0.5 rounded-full text-[11px] font-bold bg-ember-100 dark:bg-ember-900/60 text-ember-800 dark:text-ember-200 tabular-nums">
                        {item.badge}
                      </span>
                    )}
                  </>
                )}
              </NavLink>
            );
          })}
        </nav>
      </div>

      {/* Bottom status section inspired by BMS channel switcher */}
      <div className="p-3 border-t border-border space-y-3 bg-muted/20">
        {!collapsed && (
          <div className="p-2.5 rounded-lg bg-card border border-border space-y-1.5 text-xs">
            <div className="flex items-center justify-between text-muted-foreground">
              <span className="flex items-center gap-1.5">
                <Server className="w-3.5 h-3.5 text-muted-foreground" />
                <span className="font-semibold text-foreground">Services</span>
              </span>
              <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-bold">
                ONLINE
              </span>
            </div>

            <div className="flex items-center justify-between text-[11px] font-mono pt-1 border-t border-border/50">
              <span className="text-muted-foreground">HTTP</span>
              <span className="text-foreground font-semibold flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                :4010
              </span>
            </div>

            <div className="flex items-center justify-between text-[11px] font-mono">
              <span className="text-muted-foreground">SMTP</span>
              <span className="text-foreground font-semibold flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                :1025
              </span>
            </div>
          </div>
        )}

        <div className="flex items-center justify-between px-1">
          <button
            type="button"
            onClick={toggleTheme}
            className="p-2 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            title={isDark ? "Switch to light theme" : "Switch to dark theme"}
            aria-label={isDark ? "Switch to light theme" : "Switch to dark theme"}
          >
            {isDark ? (
              <Sun className="w-4 h-4 text-amber-500" />
            ) : (
              <Moon className="w-4 h-4 text-neutral-600" />
            )}
          </button>

          {!collapsed && (
            <span className="text-[11px] text-muted-foreground font-mono">mocksms v0.1.0</span>
          )}
        </div>
      </div>
    </aside>
  );
};
