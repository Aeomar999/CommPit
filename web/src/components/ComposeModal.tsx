import {
  AlertCircle,
  CheckCircle2,
  KeyRound,
  Mail,
  MessageSquare,
  Send,
  Sliders,
  Sparkles,
  X,
} from "lucide-react";
import type React from "react";
import { useState } from "react";
import { messagesApi, verificationsApi } from "../lib/api";
import { cn } from "../lib/utils";

interface ComposeModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const countryCodes = [
  { flag: "🇬🇭", code: "+233", country: "Ghana" },
  { flag: "🇺🇸", code: "+1", country: "United States" },
  { flag: "🇬🇧", code: "+44", country: "United Kingdom" },
  { flag: "🇳🇬", code: "+234", country: "Nigeria" },
  { flag: "🇰🇪", code: "+254", country: "Kenya" },
  { flag: "🇩🇪", code: "+49", country: "Germany" },
  { flag: "🇨🇦", code: "+1", country: "Canada" },
];

const sampleTemplates = [
  {
    label: "OTP Code",
    text: "Your security code is 482913. Do not share this code with anyone.",
  },
  {
    label: "Welcome Alert",
    text: "Welcome to mocksms! Your account has been created successfully.",
  },
  {
    label: "Order Update",
    text: "Order #39201 has been confirmed and is out for delivery.",
  },
];

export const ComposeModal: React.FC<ComposeModalProps> = ({ isOpen, onClose, onSuccess }) => {
  const [channel, setChannel] = useState<"sms" | "email" | "otp">("sms");
  const [selectedCountry, setSelectedCountry] = useState(countryCodes[0]);
  const [phoneNumber, setPhoneNumber] = useState("241234567");
  const [fromSender, setFromSender] = useState("MOCKSMS");
  const [emailTo, setEmailTo] = useState("developer@example.com");
  const [emailFrom, setEmailFrom] = useState("notifications@app.local");
  const [subject, setSubject] = useState("Welcome to mocksms");
  const [body, setBody] = useState("Your verification code is 482913. Valid for 10 minutes.");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  // Failure simulation toggle
  const [simulateFailure, setSimulateFailure] = useState(false);

  if (!isOpen) return null;

  // Real-time GSM-7 vs UCS-2 character counting
  const isUnicode = Array.from(body).some((char) => char.charCodeAt(0) > 127);
  const segments = Math.max(1, Math.ceil(body.length / (isUnicode ? 67 : 153)));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);
    setSuccess(false);

    try {
      if (channel === "sms") {
        const fullTo = simulateFailure
          ? "+15005550001" // Magic failure number from simulator
          : `${selectedCountry.code}${phoneNumber.replace(/^0+/, "")}`;

        await messagesApi.sendSMS({
          to: fullTo,
          from: fromSender || "MOCKSMS",
          body,
        });
      } else if (channel === "email") {
        await messagesApi.sendEmail({
          to: emailTo,
          from: emailFrom,
          subject,
          text: body,
          html: `<div style="font-family: sans-serif; padding: 20px;"><h2>${subject}</h2><p>${body}</p></div>`,
        });
      } else {
        // Verification OTP
        const fullTo = `${selectedCountry.code}${phoneNumber.replace(/^0+/, "")}`;
        await verificationsApi.create({
          to: fullTo,
          channel: "sms",
          code_length: 6,
          ttl_seconds: 600,
        });
      }

      setSuccess(true);
      setTimeout(() => {
        setIsSubmitting(false);
        onSuccess();
        onClose();
      }, 800);
    } catch (err: unknown) {
      setIsSubmitting(false);
      setError(err instanceof Error ? err.message : "Failed to send message");
    }
  };

  return (
    <div
      aria-modal="true"
      aria-labelledby="compose-modal-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-neutral-950/50 backdrop-blur-sm animate-in fade-in duration-150"
    >
      <div className="relative w-full max-w-lg rounded-xl bg-card border border-border shadow-2xl overflow-hidden animate-in zoom-in-95 duration-150">
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-muted/30">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-ember-100 dark:bg-ember-950/60 flex items-center justify-center text-ember-600 dark:text-ember-400">
              <Sparkles className="w-4 h-4" />
            </div>
            <div>
              <h2 id="compose-modal-title" className="text-base font-semibold text-foreground">
                Compose Test Message
              </h2>
              <p className="text-xs text-muted-foreground">
                Inject messages directly into your local sandbox
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            aria-label="Close dialog"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Channel Selection Segmented Pills */}
        <div className="px-6 pt-4">
          <div className="grid grid-cols-3 gap-1 p-1 bg-muted rounded-lg text-xs font-medium">
            <button
              type="button"
              onClick={() => setChannel("sms")}
              className={cn(
                "flex items-center justify-center gap-1.5 py-2 rounded-md transition-all",
                channel === "sms"
                  ? "bg-card text-foreground shadow-sm font-semibold"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <MessageSquare className="w-3.5 h-3.5 text-ember-600" />
              <span>Bulk SMS</span>
            </button>
            <button
              type="button"
              onClick={() => setChannel("email")}
              className={cn(
                "flex items-center justify-center gap-1.5 py-2 rounded-md transition-all",
                channel === "email"
                  ? "bg-card text-foreground shadow-sm font-semibold"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <Mail className="w-3.5 h-3.5 text-blue-600" />
              <span>Email</span>
            </button>
            <button
              type="button"
              onClick={() => setChannel("otp")}
              className={cn(
                "flex items-center justify-center gap-1.5 py-2 rounded-md transition-all",
                channel === "otp"
                  ? "bg-card text-foreground shadow-sm font-semibold"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <KeyRound className="w-3.5 h-3.5 text-emerald-600" />
              <span>Verify OTP</span>
            </button>
          </div>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="p-3 rounded-md bg-red-50 dark:bg-red-950/40 border border-red-200 dark:border-red-900/60 text-xs text-red-700 dark:text-red-300 flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {success && (
            <div className="p-3 rounded-md bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-900/60 text-xs text-emerald-700 dark:text-emerald-300 flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 shrink-0" />
              <span>Message dispatched to sandbox!</span>
            </div>
          )}

          {/* Recipient Input */}
          {channel !== "email" ? (
            <div className="space-y-1.5">
              <label htmlFor="phone-number" className="text-xs font-semibold text-foreground">
                Recipient Phone Number
              </label>
              <div className="flex gap-2">
                <select
                  value={selectedCountry.code}
                  onChange={(e) => {
                    const found = countryCodes.find((c) => c.code === e.target.value);
                    if (found) setSelectedCountry(found);
                  }}
                  className="px-2.5 py-2 rounded-md border border-border bg-card text-foreground text-xs font-medium focus:outline-none focus:ring-2 focus:ring-ember-500"
                  aria-label="Country Code"
                >
                  {countryCodes.map((c) => (
                    <option key={`${c.code}-${c.country}`} value={c.code}>
                      {c.flag} {c.code} ({c.country})
                    </option>
                  ))}
                </select>
                <input
                  id="phone-number"
                  type="tel"
                  value={phoneNumber}
                  onChange={(e) => setPhoneNumber(e.target.value)}
                  placeholder="24 123 4567"
                  className="flex-1 px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs font-mono focus:outline-none focus:ring-2 focus:ring-ember-500"
                  required
                />
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label htmlFor="email-to" className="text-xs font-semibold text-foreground">
                  Recipient Email
                </label>
                <input
                  id="email-to"
                  type="email"
                  value={emailTo}
                  onChange={(e) => setEmailTo(e.target.value)}
                  className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500"
                  required
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="email-from" className="text-xs font-semibold text-foreground">
                  Sender From
                </label>
                <input
                  id="email-from"
                  type="email"
                  value={emailFrom}
                  onChange={(e) => setEmailFrom(e.target.value)}
                  className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500"
                  required
                />
              </div>
            </div>
          )}

          {channel === "sms" && (
            <div className="space-y-1.5">
              <label htmlFor="sender-id" className="text-xs font-semibold text-foreground">
                Sender ID / From
              </label>
              <input
                id="sender-id"
                type="text"
                value={fromSender}
                onChange={(e) => setFromSender(e.target.value)}
                placeholder="MOCKSMS"
                className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500"
              />
            </div>
          )}

          {channel === "email" && (
            <div className="space-y-1.5">
              <label htmlFor="email-subject" className="text-xs font-semibold text-foreground">
                Subject
              </label>
              <input
                id="email-subject"
                type="text"
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                placeholder="Email Subject"
                className="w-full px-3 py-2 rounded-md border border-border bg-card text-foreground text-xs focus:outline-none focus:ring-2 focus:ring-ember-500"
                required
              />
            </div>
          )}

          {channel !== "otp" && (
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label htmlFor="message-body" className="text-xs font-semibold text-foreground">
                  Message Content
                </label>
                <div className="flex gap-1.5">
                  {sampleTemplates.map((tpl) => (
                    <button
                      key={tpl.label}
                      type="button"
                      onClick={() => setBody(tpl.text)}
                      className="text-[11px] px-2 py-0.5 rounded bg-muted hover:bg-neutral-200 dark:hover:bg-neutral-800 text-muted-foreground hover:text-foreground transition-colors"
                    >
                      {tpl.label}
                    </button>
                  ))}
                </div>
              </div>
              <textarea
                id="message-body"
                rows={3}
                value={body}
                onChange={(e) => setBody(e.target.value)}
                placeholder="Type your message content here..."
                className="w-full p-3 rounded-md border border-border bg-card text-foreground text-xs leading-relaxed focus:outline-none focus:ring-2 focus:ring-ember-500"
                required
              />

              {channel === "sms" && (
                <div className="flex items-center justify-between text-[11px] text-muted-foreground font-mono">
                  <span>
                    {body.length} chars · {segments} segment{segments > 1 ? "s" : ""} (
                    {isUnicode ? "UCS-2 Unicode" : "GSM-7 standard"})
                  </span>
                  <span>Max 10 segments</span>
                </div>
              )}
            </div>
          )}

          {/* Simulation Options */}
          <div className="p-3 rounded-md bg-muted/40 border border-border space-y-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Sliders className="w-3.5 h-3.5 text-muted-foreground" />
                <span className="text-xs font-medium text-foreground">
                  Simulate Delivery Failure
                </span>
              </div>
              <input
                type="checkbox"
                id="sim-fail"
                checked={simulateFailure}
                onChange={(e) => setSimulateFailure(e.target.checked)}
                className="h-4 w-4 rounded border-border text-ember-600 focus:ring-ember-500"
              />
            </div>
            {simulateFailure && (
              <p className="text-[11px] text-amber-700 dark:text-amber-300">
                Will route to magic pattern +15005550001 to simulate carrier rejection.
              </p>
            )}
          </div>

          {/* Modal Actions */}
          <div className="flex items-center justify-end gap-3 pt-3 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-md border border-border bg-card hover:bg-muted text-xs font-medium text-foreground transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex items-center gap-2 px-5 py-2 rounded-md bg-ember-500 hover:bg-ember-600 text-ember-950 font-semibold text-xs shadow-sm transition-all disabled:opacity-50"
            >
              <Send className="w-3.5 h-3.5" />
              <span>{isSubmitting ? "Sending..." : "Send Test Message"}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
