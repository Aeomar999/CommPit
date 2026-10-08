import {
  Check,
  CheckCircle2,
  Circle,
  Copy,
  KeyRound,
  Mail,
  Smartphone,
  Sparkles,
  Terminal,
  X,
} from "lucide-react";
import type React from "react";
import { useState } from "react";
import { cn } from "../lib/utils";

interface Step {
  id: string;
  title: string;
  description: string;
  command?: string;
  icon: React.ElementType;
}

const steps: Step[] = [
  {
    id: "email",
    title: "1. Send your first email via local SMTP",
    description:
      "Configure your app to point SMTP to localhost port 1025 with no TLS or credentials.",
    command: "swaks --to user@example.com --from test@app.local --server 127.0.0.1:1025",
    icon: Mail,
  },
  {
    id: "sms",
    title: "2. Send an SMS using the native REST API",
    description: "Make a quick POST request to /api/v1/sms to see a simulated text appear.",
    command: `curl -X POST http://127.0.0.1:4010/api/v1/sms \\\n  -H "Content-Type: application/json" \\\n  -d '{"to": "+233241234567", "from": "ACME", "body": "Your code is 482913"}'`,
    icon: Smartphone,
  },
  {
    id: "sdk",
    title: "3. Point Twilio or Termii SDK at mocksms",
    description:
      "Replace the API base URL in your existing SDK configuration with http://127.0.0.1:4010.",
    command: 'export TWILIO_BASE_URL="http://127.0.0.1:4010"',
    icon: Terminal,
  },
  {
    id: "otp",
    title: "4. Read the latest OTP code in automated tests",
    description:
      "Fetch the most recent verification code synchronously without polling or parsing mail.",
    command: "curl http://127.0.0.1:4010/api/v1/otp/latest?to=%2B233241234567",
    icon: KeyRound,
  },
];

interface OnboardingChecklistProps {
  completedStepIds?: string[];
  onDismiss?: () => void;
  className?: string;
}

export const OnboardingChecklist: React.FC<OnboardingChecklistProps> = ({
  completedStepIds = [],
  onDismiss,
  className,
}) => {
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const completedCount = completedStepIds.length;

  const handleCopy = (id: string, text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 1500);
  };

  return (
    <div
      className={cn("rounded-lg bg-card border border-border shadow-sm p-5 sm:p-6 mb-6", className)}
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-ember-100 dark:bg-ember-950/60 flex items-center justify-center text-ember-600 dark:text-ember-400">
            <Sparkles className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-base font-bold text-foreground">Get started with mocksms</h3>
            <p className="text-xs text-muted-foreground">
              {completedCount} of {steps.length} completed · Sandbox is ready and listening
            </p>
          </div>
        </div>

        {onDismiss && (
          <button
            type="button"
            onClick={onDismiss}
            className="p-1 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            title="Dismiss checklist"
            aria-label="Dismiss checklist"
          >
            <X className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Progress bar */}
      <div className="w-full h-1.5 rounded-full bg-neutral-100 dark:bg-neutral-800 mb-5 overflow-hidden">
        <div
          className="h-full bg-ember-500 transition-all duration-500 rounded-full"
          style={{ width: `${(completedCount / steps.length) * 100}%` }}
        />
      </div>

      {/* Steps List */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {steps.map((step) => {
          const isDone = completedStepIds.includes(step.id);
          const Icon = step.icon;

          return (
            <div
              key={step.id}
              className={cn(
                "flex flex-col justify-between p-3.5 rounded-lg border transition-all",
                isDone
                  ? "bg-neutral-50/50 dark:bg-neutral-900/30 border-neutral-200 dark:border-neutral-800 opacity-80"
                  : "bg-card border-border hover:border-ember-300 dark:hover:border-ember-800/80"
              )}
            >
              <div>
                <div className="flex items-center justify-between gap-2 mb-1.5">
                  <div className="flex items-center gap-2">
                    {isDone ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400 shrink-0" />
                    ) : (
                      <Circle className="w-4 h-4 text-muted-foreground shrink-0" />
                    )}
                    <span className="text-xs font-semibold text-foreground">{step.title}</span>
                  </div>
                  <Icon className="w-3.5 h-3.5 text-muted-foreground" />
                </div>
                <p className="text-xs text-muted-foreground mb-3 leading-relaxed">
                  {step.description}
                </p>
              </div>

              {step.command && (
                <div className="relative group mt-auto">
                  <pre className="p-2.5 rounded-md bg-neutral-900 text-neutral-100 font-mono text-[11px] overflow-x-auto whitespace-pre-wrap leading-tight select-all">
                    {step.command}
                  </pre>
                  <button
                    type="button"
                    onClick={() => handleCopy(step.id, step.command!)}
                    className="absolute top-2 right-2 p-1 rounded bg-neutral-800/80 hover:bg-neutral-700 text-neutral-300 transition-colors"
                    title="Copy command"
                    aria-label="Copy command"
                  >
                    {copiedId === step.id ? (
                      <Check className="w-3 h-3 text-emerald-400" />
                    ) : (
                      <Copy className="w-3 h-3" />
                    )}
                  </button>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};
