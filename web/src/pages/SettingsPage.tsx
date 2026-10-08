import { Check, Globe, Info, KeyRound, Save, Shield, Trash2 } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { projectsApi } from "../lib/api";
import type { Project } from "../types";

export function SettingsPage() {
  const [saved, setSaved] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);
  const [linkProject, setLinkProject] = useState("");
  const [linkProvider, setLinkProvider] = useState("twilio");
  const [linkKey, setLinkKey] = useState("");
  const [linkStatus, setLinkStatus] = useState<{ ok: boolean; text: string } | null>(null);
  const [linking, setLinking] = useState(false);

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  useEffect(() => {
    projectsApi
      .list()
      .then((resp) => {
        setProjects(resp.projects);
        setLinkProject((current) => current || resp.projects[0]?.id || "");
      })
      .catch(() => setProjects([]));
  }, []);

  const handleLink = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!linkProject || !linkKey.trim()) {
      setLinkStatus({ ok: false, text: "Choose a project and enter a credential key." });
      return;
    }
    setLinking(true);
    setLinkStatus(null);
    try {
      await projectsApi.linkCredential(linkProject, {
        provider: linkProvider,
        key: linkKey.trim(),
      });
      setLinkStatus({ ok: true, text: `Credential linked to ${linkProject}.` });
      setLinkKey("");
    } catch (err) {
      setLinkStatus({
        ok: false,
        text: err instanceof Error ? err.message : "Linking failed.",
      });
    } finally {
      setLinking(false);
    }
  };

  return (
    <div className="space-y-6 max-w-4xl">
      <div>
        <h2 className="text-xl font-bold tracking-tight text-foreground">
          Settings & Configuration
        </h2>
        <p className="text-xs text-muted-foreground mt-0.5">
          Configure project scoping, allowed hosts, and retention prune policies
        </p>
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* General Project Section */}
        <div className="rounded-xl bg-card border border-border shadow-xs p-5 sm:p-6 space-y-4">
          <div className="flex items-center gap-2 pb-3 border-b border-border">
            <Globe className="w-4 h-4 text-ember-600" />
            <h3 className="text-sm font-semibold text-foreground">General Configuration</h3>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <label htmlFor="project-name" className="text-xs font-semibold text-foreground">
                Project Name
              </label>
              <input
                id="project-name"
                type="text"
                defaultValue="default"
                className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500 font-mono"
              />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="timezone-select" className="text-xs font-semibold text-foreground">
                Display Timezone
              </label>
              <select
                id="timezone-select"
                defaultValue="UTC"
                className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500"
              >
                <option value="UTC">UTC (Coordinated Universal Time)</option>
                <option value="Africa/Accra">Africa/Accra (GMT+0)</option>
                <option value="America/New_York">America/New_York (EST)</option>
                <option value="Europe/London">Europe/London (BST)</option>
              </select>
            </div>
          </div>
        </div>

        {/* Security & Access Section */}
        <div className="rounded-xl bg-card border border-border shadow-xs p-5 sm:p-6 space-y-4">
          <div className="flex items-center gap-2 pb-3 border-b border-border">
            <Shield className="w-4 h-4 text-ember-600" />
            <h3 className="text-sm font-semibold text-foreground">Security & Allowed Hosts</h3>
          </div>

          <div className="space-y-4">
            <div className="space-y-1.5">
              <label htmlFor="allowed-hosts" className="text-xs font-semibold text-foreground">
                Allowed Host Headers (DNS Rebinding Protection)
              </label>
              <textarea
                id="allowed-hosts"
                rows={3}
                defaultValue="127.0.0.1&#10;localhost"
                className="w-full p-3 rounded-md border border-border bg-card text-foreground font-mono text-xs focus:outline-none focus:ring-2 focus:ring-ember-500 leading-relaxed"
              />
              <p className="text-[11px] text-muted-foreground">
                One host per line. Requests with other Host headers return HTTP 421 Misdirected
                Request.
              </p>
            </div>

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="require-x-mocksms"
                defaultChecked
                className="h-4 w-4 rounded border-border text-ember-600 focus:ring-ember-500"
              />
              <label htmlFor="require-x-mocksms" className="text-xs text-foreground select-none">
                Require{" "}
                <code className="font-mono bg-muted px-1 py-0.5 rounded text-[11px]">
                  X-Mocksms
                </code>{" "}
                header on API requests (CSRF mitigation)
              </label>
            </div>
          </div>
        </div>

        {/* Data Retention Section */}
        <div className="rounded-xl bg-card border border-border shadow-xs p-5 sm:p-6 space-y-4">
          <div className="flex items-center gap-2 pb-3 border-b border-border">
            <Trash2 className="w-4 h-4 text-ember-600" />
            <h3 className="text-sm font-semibold text-foreground">Data Retention & Auto-Pruning</h3>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <label htmlFor="message-ttl" className="text-xs font-semibold text-foreground">
                Message TTL
              </label>
              <select
                id="message-ttl"
                defaultValue="30d"
                className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500"
              >
                <option value="24h">24 hours</option>
                <option value="7d">7 days</option>
                <option value="30d">30 days</option>
                <option value="90d">90 days</option>
              </select>
            </div>
            <div className="space-y-1.5">
              <label htmlFor="prune-interval" className="text-xs font-semibold text-foreground">
                Prune Interval
              </label>
              <select
                id="prune-interval"
                defaultValue="1h"
                className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500"
              >
                <option value="15m">15 minutes</option>
                <option value="1h">1 hour</option>
                <option value="6h">6 hours</option>
              </select>
            </div>
          </div>
        </div>

        {/* Linked Credentials Section */}
        <div className="rounded-xl bg-card border border-border shadow-xs p-5 sm:p-6 space-y-4">
          <div className="flex items-center gap-2 pb-3 border-b border-border">
            <KeyRound className="w-4 h-4 text-ember-600" />
            <h3 className="text-sm font-semibold text-foreground">Linked Credentials</h3>
          </div>

          <p className="text-[11px] text-muted-foreground">
            Map several provider credentials (Twilio, Termii, SMTP, native keys) onto one project,
            so traffic from any of them lands in the same inbox.
          </p>

          <form onSubmit={handleLink} className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <label htmlFor="link-project" className="text-xs font-semibold text-foreground">
                Project
              </label>
              <select
                id="link-project"
                value={linkProject}
                onChange={(e) => setLinkProject(e.target.value)}
                className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500 font-mono"
              >
                {projects.map((project) => (
                  <option key={project.id} value={project.id}>
                    {project.name} ({project.id})
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <label htmlFor="link-provider" className="text-xs font-semibold text-foreground">
                Provider
              </label>
              <select
                id="link-provider"
                value={linkProvider}
                onChange={(e) => setLinkProvider(e.target.value)}
                className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500"
              >
                <option value="twilio">twilio</option>
                <option value="termii">termii</option>
                <option value="smtp">smtp</option>
                <option value="native">native</option>
              </select>
            </div>
            <div className="space-y-1.5 md:col-span-2">
              <label htmlFor="link-key" className="text-xs font-semibold text-foreground">
                Credential Key
              </label>
              <input
                id="link-key"
                type="text"
                value={linkKey}
                onChange={(e) => setLinkKey(e.target.value)}
                placeholder="Account SID, api_key, SMTP username or Bearer key"
                className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500 font-mono"
              />
            </div>
            <div className="md:col-span-2 flex items-center justify-end gap-3">
              {linkStatus && (
                <output
                  className={
                    linkStatus.ok
                      ? "text-xs text-emerald-600 dark:text-emerald-400 font-medium"
                      : "text-xs text-red-600 dark:text-red-400 font-medium"
                  }
                >
                  {linkStatus.text}
                </output>
              )}
              <button
                type="submit"
                disabled={linking}
                className="inline-flex items-center gap-2 px-5 py-2 rounded-md bg-ember-500 hover:bg-ember-600 text-ember-950 font-semibold text-xs shadow-sm transition-all active:scale-[0.98] disabled:opacity-50"
              >
                <KeyRound className="w-4 h-4" />
                <span>{linking ? "Linking…" : "Link Credential"}</span>
              </button>
            </div>
          </form>
        </div>

        {/* BMS-inspired Heads-up info card */}
        <div className="p-4 rounded-xl bg-ember-50/60 dark:bg-ember-950/20 border border-ember-200/80 dark:border-ember-900/40 text-xs text-ember-900 dark:text-ember-200 flex items-start gap-3">
          <Info className="w-4 h-4 text-ember-600 shrink-0 mt-0.5" />
          <div className="space-y-1">
            <span className="font-bold block">Local-First Sandbox Guarantee</span>
            <p className="text-[11px] leading-relaxed text-ember-800 dark:text-ember-300">
              Settings are stored in the local SQLite database. Messages and verification codes are
              never transmitted to third parties or remote servers.
            </p>
          </div>
        </div>

        {/* Submit Button */}
        <div className="flex items-center justify-end gap-3 pt-2">
          {saved && (
            <span className="inline-flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400 font-medium">
              <Check className="w-4 h-4" />
              Settings saved!
            </span>
          )}
          <button
            type="submit"
            className="inline-flex items-center gap-2 px-5 py-2 rounded-md bg-ember-500 hover:bg-ember-600 text-ember-950 font-semibold text-xs shadow-sm transition-all active:scale-[0.98]"
          >
            <Save className="w-4 h-4" />
            <span>Save Settings</span>
          </button>
        </div>
      </form>
    </div>
  );
}
