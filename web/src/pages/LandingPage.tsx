import { motion } from "framer-motion";
import {
  ArrowRight,
  Check,
  CheckCircle2,
  Code2,
  Copy,
  Cpu,
  ExternalLink,
  Flame,
  Globe,
  HardDrive,
  KeyRound,
  Lock,
  Mail,
  MessageSquare,
  Phone,
  RefreshCw,
  Send,
  ShieldCheck,
  Smartphone,
  Terminal,
} from "lucide-react";
import { useId, useState } from "react";
import { Link } from "react-router-dom";
import { StoryScroll } from "../components/StoryScroll";
import { cn } from "../lib/utils";

// Supported install methods for quick start
interface InstallTab {
  id: "brew" | "docker" | "curl" | "scoop";
  label: string;
  command: string;
}

const installTabs: InstallTab[] = [
  {
    id: "brew",
    label: "Homebrew",
    command: "brew install mocksms/tap/mocksms && mocksms serve",
  },
  {
    id: "docker",
    label: "Docker",
    command: "docker run -d -p 4010:4010 -p 1025:1025 ghcr.io/mocksms/mocksms:latest",
  },
  {
    id: "curl",
    label: "Linux / macOS",
    command: "curl -fsSL https://raw.githubusercontent.com/Aeomar999/CommPit/main/install.sh | sh",
  },
  {
    id: "scoop",
    label: "Windows Scoop",
    command: "scoop bucket add mocksms && scoop install mocksms",
  },
];

// Interactive Code Snippets for SDK Redirect
interface CodeSnippet {
  id: "curl" | "node" | "python" | "go" | "smtp";
  label: string;
  filename: string;
  language: string;
  code: string;
}

const codeSnippets: CodeSnippet[] = [
  {
    id: "curl",
    label: "cURL",
    filename: "send-sms.sh",
    language: "bash",
    code: `curl -X POST http://127.0.0.1:4010/api/v1/sms \\
  -H "Authorization: Bearer mock_key_dev" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "+233241234567",
    "from": "MOCKSMS",
    "body": "Your verification code is 5892. Valid for 10 minutes."
  }'`,
  },
  {
    id: "node",
    label: "Twilio Node.js",
    filename: "auth-service.ts",
    language: "typescript",
    code: `import twilio from "twilio";

// Credentials are dummy in local sandbox mode
const client = twilio("AC_mock_sandbox", "token_mock");

// Simply redirect client base URL to local mocksms instance
client.api.baseUrl = "http://127.0.0.1:4010";

export async function sendVerification(to: string, code: string) {
  return await client.messages.create({
    to,
    from: "MOCKSMS",
    body: \`Your security verification code is \${code}.\`
  });
}`,
  },
  {
    id: "python",
    label: "Twilio Python",
    filename: "notifications.py",
    language: "python",
    code: `from twilio.rest import Client

# Local sandbox client with dummy credentials
client = Client("AC_mock_sandbox", "token_mock")
client.http_client.base_url = "http://127.0.0.1:4010"

def dispatch_login_otp(phone: str, code: str):
    message = client.messages.create(
        to=phone,
        from_="MOCKSMS",
        body=f"Your security verification code is {code}."
    )
    return message.sid`,
  },
  {
    id: "go",
    label: "Go SDK",
    filename: "sms.go",
    language: "go",
    code: `package main

import (
    "context"
    "github.com/twilio/twilio-go"
    openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

func SendSMS(ctx context.Context, to, body string) error {
    client := twilio.NewRestClient()
    client.BaseURL = "http://127.0.0.1:4010"

    params := &openapi.CreateMessageParams{
        To:   &to,
        From: twilio.String("MOCKSMS"),
        Body: &body,
    }
    _, err := client.Api.CreateMessage(params)
    return err
}`,
  },
  {
    id: "smtp",
    label: "SMTP (Nodemailer)",
    filename: "mailer.ts",
    language: "typescript",
    code: `import nodemailer from "nodemailer";

// Points to embedded mocksms SMTP daemon on port 1025
export const mailer = nodemailer.createTransport({
  host: "127.0.0.1",
  port: 1025,
  secure: false
});

await mailer.sendMail({
  from: '"Auth Security" <auth@local.app>',
  to: "developer@acme.dev",
  subject: "Confirm your login request",
  html: "<p>Your one-time code is <b>5892</b></p>"
});`,
  },
];

// Magic number pattern test cases
interface MagicNumberCase {
  number: string;
  label: string;
  simulatedError: string;
  statusCode: number;
}

const magicNumberCases: MagicNumberCase[] = [
  {
    number: "+233 24 000 999901",
    label: "Invalid Destination",
    simulatedError: "Provider Error 21211: 'To' number is not a valid mobile handset",
    statusCode: 400,
  },
  {
    number: "+233 24 000 999902",
    label: "Rate Limit Exceeded",
    simulatedError: "Provider Error 20429: Account queue capacity exceeded (429)",
    statusCode: 429,
  },
  {
    number: "+233 24 000 999903",
    label: "Carrier Blacklist",
    simulatedError: "Provider Error 30007: Filtered by destination carrier spam firewall",
    statusCode: 400,
  },
  {
    number: "+233 24 000 999904",
    label: "Unreachable Route",
    simulatedError: "Provider Error 30008: Unknown destination carrier route degradation",
    statusCode: 502,
  },
];

export function LandingPage() {
  const volumeSliderId = useId();
  const composerTextId = useId();

  // Install command switcher state
  const [activeInstallTab, setActiveInstallTab] = useState<InstallTab["id"]>("brew");
  const [copiedInstall, setCopiedInstall] = useState(false);

  // Code editor tabs state
  const [activeCodeSnippet, setActiveCodeSnippet] = useState<CodeSnippet["id"]>("curl");
  const [copiedSnippet, setCopiedSnippet] = useState(false);

  // Volume slider calculation state
  const [volume, setVolume] = useState<number>(45000);
  const [pricingInterval, setPricingInterval] = useState<"dev" | "ci" | "prod">("dev");

  // Interactive Live Hero OTP demo
  const [liveOtp, setLiveOtp] = useState("5892");
  const [recipientNumber, setRecipientNumber] = useState("+233 24 123 4567");
  const [demoPulsing, setDemoPulsing] = useState(false);
  const [activeDemoFeed, setActiveDemoFeed] = useState<
    Array<{ id: string; provider: string; to: string; code: string; time: string }>
  >([
    {
      id: "msg_01J98X",
      provider: "Twilio",
      to: "+233 24 123 4567",
      code: "5892",
      time: "Just now",
    },
    {
      id: "msg_01J97P",
      provider: "Termii",
      to: "+234 80 987 6543",
      code: "8319",
      time: "14s ago",
    },
    {
      id: "msg_01J96K",
      provider: "SMTP",
      to: "qa-dev@acme.local",
      code: "1490",
      time: "1m ago",
    },
  ]);

  // Interactive SMS Composer live calculator state
  const [composerText, setComposerText] = useState(
    "Your verification code is {{code}}. Valid for 10 minutes. Do not share this code."
  );
  const [composerMode, setComposerMode] = useState<"sms" | "smtp" | "inbound">("sms");

  // Magic numbers edge case interactive state
  const [selectedMagicIndex, setSelectedMagicIndex] = useState(0);

  const currentInstall = installTabs.find((t) => t.id === activeInstallTab) || installTabs[0];
  const currentSnippet = codeSnippets.find((s) => s.id === activeCodeSnippet) || codeSnippets[0];
  const currentMagic = magicNumberCases[selectedMagicIndex];

  // Character and segment calculation
  const isUnicode = Array.from(composerText).some((c) => c.charCodeAt(0) > 127);
  const charLimit = isUnicode ? 70 : 160;
  const segmentCount = Math.max(1, Math.ceil(composerText.length / charLimit));

  const handleCopyInstall = () => {
    navigator.clipboard.writeText(currentInstall.command);
    setCopiedInstall(true);
    setTimeout(() => setCopiedInstall(false), 1500);
  };

  const handleCopySnippet = () => {
    navigator.clipboard.writeText(currentSnippet.code);
    setCopiedSnippet(true);
    setTimeout(() => setCopiedSnippet(false), 1500);
  };

  const handleTriggerSimulate = (providerName: string) => {
    setDemoPulsing(true);
    const newCode = String(Math.floor(1000 + Math.random() * 9000));
    const randomSuffix = String(Math.floor(1000 + Math.random() * 9000));
    const newPhone = `+233 24 ${randomSuffix.slice(0, 3)} ${randomSuffix.slice(3)}`;

    setLiveOtp(newCode);
    setRecipientNumber(newPhone);

    setActiveDemoFeed((prev) => [
      {
        id: `msg_${Math.random().toString(36).substring(2, 8).toUpperCase()}`,
        provider: providerName,
        to: newPhone,
        code: newCode,
        time: "Just now",
      },
      ...prev.slice(0, 2),
    ]);

    setTimeout(() => setDemoPulsing(false), 500);
  };

  // Dollar savings calculations vs real commercial gateways
  const twilioSaved = (volume * 0.0079).toFixed(2);
  const termiiSaved = (volume * 0.0052).toFixed(2);

  return (
    <div className="min-h-screen bg-white text-[#0F172A] font-sans antialiased selection:bg-orange-100 selection:text-orange-900">
      {/* 1. TOP NAVIGATION (World-Class Glassmorphic Bar) */}
      <header className="sticky top-0 z-50 bg-white/90 backdrop-blur-md border-b border-slate-200/80 shadow-[0_1px_3px_rgba(0,0,0,0.02)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-18 flex items-center justify-between">
          {/* Logo with crisp glow badge */}
          <Link to="/inbox" className="group flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-orange-600 to-amber-500 flex items-center justify-center text-white shadow-md shadow-orange-500/20 group-hover:scale-110 transition-transform duration-300">
              <MessageSquare className="w-5 h-5 fill-current" />
            </div>
            <div className="flex items-baseline">
              <span className="text-xl font-black tracking-tight text-slate-900">mock</span>
              <span className="text-xl font-black tracking-tight text-orange-500">sms</span>
            </div>
          </Link>

          {/* Navigation Links with pill hover states */}
          <nav className="hidden lg:flex items-center gap-1 text-sm font-semibold text-slate-600">
            <a
              href="#features"
              className="px-3.5 py-1.5 rounded-full hover:text-slate-900 hover:bg-slate-100 transition-colors"
            >
              Features
            </a>
            <a
              href="#adapters"
              className="px-3.5 py-1.5 rounded-full hover:text-slate-900 hover:bg-slate-100 transition-colors"
            >
              Adapters
            </a>
            <a
              href="#composer"
              className="px-3.5 py-1.5 rounded-full hover:text-slate-900 hover:bg-slate-100 transition-colors"
            >
              Sandbox Playground
            </a>
            <a
              href="#api"
              className="px-3.5 py-1.5 rounded-full hover:text-slate-900 hover:bg-slate-100 transition-colors"
            >
              API Reference
            </a>
            <a
              href="#savings"
              className="px-3.5 py-1.5 rounded-full hover:text-slate-900 hover:bg-slate-100 transition-colors"
            >
              Cost Savings
            </a>
          </nav>

          {/* Right Header CTAs */}
          <div className="flex items-center gap-3">
            <div className="hidden sm:inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-50 text-emerald-800 border border-emerald-200/80 text-xs font-semibold">
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
              <span>Sandbox Active (0ms)</span>
            </div>

            <a
              href="https://github.com/Aeomar999/CommPit"
              target="_blank"
              rel="noopener noreferrer"
              className="p-2 text-slate-500 hover:text-slate-900 hover:bg-slate-100 rounded-full transition-colors"
              aria-label="View on GitHub"
            >
              <ExternalLink className="w-4 h-4" />
            </a>

            <Link
              to="/inbox"
              className="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-full bg-gradient-to-r from-orange-500 to-amber-500 hover:from-orange-600 hover:to-amber-600 text-white font-bold text-sm shadow-md shadow-orange-500/25 transition-all duration-200 active:scale-[0.98]"
            >
              <span>Open Web Inbox</span>
              <ArrowRight className="w-4 h-4" />
            </Link>
          </div>
        </div>
      </header>

      {/* 2. HERO SECTION WITH AMBIENT GLOW & CENTERPIECE CANVAS */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        className="relative pt-12 pb-20 lg:pt-20 lg:pb-28 overflow-hidden"
      >
        {/* Subtle Ambient Radial Lighting */}
        <div
          className="absolute -top-40 left-1/2 -translate-x-1/2 w-[1000px] h-[550px] opacity-40 pointer-events-none blur-3xl"
          style={{
            background:
              "radial-gradient(ellipse at center, rgba(249, 115, 22, 0.28) 0%, rgba(245, 158, 11, 0.12) 45%, transparent 70%)",
          }}
        />

        <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
            className="max-w-3xl mx-auto lg:mx-0 space-y-6 text-center lg:text-left"
          >
            {/* Pill Tag */}
            <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-orange-50 border border-orange-200/80 text-orange-800 text-xs font-bold tracking-wide uppercase shadow-xs">
              <Flame className="w-3.5 h-3.5 text-orange-500" />
              <span>Local-First Sandbox Provider Â· Wave 1 Available</span>
            </div>

            {/* Main Headline */}
            <h1 className="text-4xl sm:text-6xl lg:text-7xl font-black tracking-tight text-slate-950 leading-[1.05]">
              The local-first <br />
              <span className="text-transparent bg-clip-text bg-gradient-to-r from-orange-500 via-amber-500 to-orange-600">
                messaging sandbox.
              </span>
            </h1>

            {/* Subtitle */}
            <p className="text-base sm:text-xl text-slate-600 max-w-2xl leading-relaxed mx-auto lg:mx-0">
              Develop and test SMS, OTP, and email flows without third-party fees, leaking customer
              phone numbers, or hitting carrier rate limits. Point your existing SDK to localhost
              and ship with confidence.
            </p>

            {/* Action Buttons */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.8, delay: 0.2, ease: [0.16, 1, 0.3, 1] }}
              className="flex flex-wrap items-center justify-center lg:justify-start gap-4 pt-2"
            >
              <Link
                to="/inbox"
                className="inline-flex items-center justify-center gap-2 px-8 py-3.5 rounded-full bg-gradient-to-r from-orange-500 to-amber-500 hover:from-orange-600 hover:to-amber-600 text-white font-bold text-sm shadow-lg shadow-orange-500/30 transition-all duration-200 hover:scale-[1.02] active:scale-[0.98]"
              >
                <span>Launch Local Console</span>
                <ArrowRight className="w-4 h-4" />
              </Link>
              <a
                href="#composer"
                className="inline-flex items-center justify-center gap-2 px-7 py-3.5 rounded-full bg-white hover:bg-slate-50 text-slate-800 font-bold text-sm border border-slate-300/80 shadow-xs transition-all duration-300 hover:scale-[1.03] active:scale-[0.97]"
              >
                <span>Try Live Composer</span>
              </a>
            </motion.div>

            {/* Quick Install Tabs Strip */}
            <div className="pt-3 max-w-xl mx-auto lg:mx-0">
              <div className="rounded-2xl bg-slate-900 border border-slate-800 shadow-xl p-3 text-left">
                <div className="flex items-center justify-between pb-2 border-b border-slate-800 px-1">
                  <div className="flex items-center gap-1">
                    {installTabs.map((tab) => (
                      <button
                        key={tab.id}
                        type="button"
                        onClick={() => setActiveInstallTab(tab.id)}
                        className={cn(
                          "px-2.5 py-1 rounded-md text-xs font-semibold transition-colors",
                          activeInstallTab === tab.id
                            ? "bg-slate-800 text-orange-400 font-bold"
                            : "text-slate-400 hover:text-white"
                        )}
                      >
                        {tab.label}
                      </button>
                    ))}
                  </div>
                  <span className="text-[11px] font-mono text-slate-500">Single Binary</span>
                </div>

                <div className="relative group pt-2 px-1">
                  <pre className="text-xs font-mono text-slate-200 overflow-x-auto whitespace-pre-wrap select-all pr-12">
                    {currentInstall.command}
                  </pre>
                  <button
                    type="button"
                    onClick={handleCopyInstall}
                    className="absolute top-2 right-1 p-1.5 rounded-md bg-slate-800 hover:bg-slate-700 text-slate-300 transition-colors"
                    aria-label="Copy install command"
                  >
                    {copiedInstall ? (
                      <Check className="w-3.5 h-3.5 text-emerald-400" />
                    ) : (
                      <Copy className="w-3.5 h-3.5" />
                    )}
                  </button>
                </div>
              </div>
            </div>

            {/* Channels & Native Adapters Strip */}
            <div
              id="adapters"
              className="pt-6 grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 border-t border-slate-200/80 text-xs font-semibold text-slate-700"
            >
              <div className="flex items-center gap-2 p-2 rounded-lg bg-slate-50 border border-slate-100">
                <MessageSquare className="w-4 h-4 text-orange-500 shrink-0" />
                <span>Twilio SMS</span>
              </div>
              <div className="flex items-center gap-2 p-2 rounded-lg bg-slate-50 border border-slate-100">
                <Phone className="w-4 h-4 text-orange-500 shrink-0" />
                <span>Termii API</span>
              </div>
              <div className="flex items-center gap-2 p-2 rounded-lg bg-slate-50 border border-slate-100">
                <KeyRound className="w-4 h-4 text-orange-500 shrink-0" />
                <span>OTP Extraction</span>
              </div>
              <div className="flex items-center gap-2 p-2 rounded-lg bg-slate-50 border border-slate-100">
                <Mail className="w-4 h-4 text-orange-500 shrink-0" />
                <span>SMTP (:1025)</span>
              </div>
              <div className="flex items-center gap-2 p-2 rounded-lg bg-slate-50 border border-slate-100">
                <Globe className="w-4 h-4 text-orange-500 shrink-0" />
                <span>Webhooks</span>
              </div>
              <div className="flex items-center gap-2 p-2 rounded-lg bg-slate-50 border border-slate-100">
                <Smartphone className="w-4 h-4 text-orange-500 shrink-0" />
                <span>Test REST API</span>
              </div>
            </div>
          </motion.div>

          {/* Large Hero Interactive Browser Showcase Window */}
          <div className="mt-14 rounded-3xl bg-slate-900/5 p-2 sm:p-4 border border-slate-200 shadow-2xl">
            <div className="rounded-2xl bg-white border border-slate-200 shadow-lg overflow-hidden">
              {/* macOS Window Chrome Bar */}
              <div className="px-4 py-3 bg-slate-100 border-b border-slate-200 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-3 h-3 rounded-full bg-[#FF5F56]" />
                  <div className="w-3 h-3 rounded-full bg-[#FFBD2E]" />
                  <div className="w-3 h-3 rounded-full bg-[#27C93F]" />
                </div>
                <div className="px-6 py-1 bg-white border border-slate-300 rounded-full text-xs font-mono text-slate-600 shadow-inner max-w-sm w-full text-center flex items-center justify-center gap-2">
                  <Lock className="w-3 h-3 text-slate-400" />
                  <span>127.0.0.1:4010/inbox</span>
                </div>
                <span className="text-[11px] font-mono text-slate-400">Local Sandbox</span>
              </div>

              {/* Showcase Body: Split-Pane Live Messaging Console */}
              <div className="p-6 sm:p-10 lg:p-12 bg-gradient-to-br from-orange-50/40 via-white to-amber-50/30">
                <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
                  {/* Left Column: Live Message Feed Simulation */}
                  <div className="lg:col-span-5 space-y-3">
                    <div className="flex items-center justify-between pb-2">
                      <span className="text-xs font-bold uppercase tracking-wider text-slate-500">
                        Live Intercept Stream
                      </span>
                      <span className="text-[11px] font-bold text-emerald-700 bg-emerald-100 px-2 py-0.5 rounded-full">
                        SSE Connected
                      </span>
                    </div>

                    <div className="space-y-2">
                      {activeDemoFeed.map((item) => (
                        <div
                          key={item.id}
                          className="p-3.5 rounded-xl bg-white border border-slate-200 shadow-xs flex items-center justify-between gap-3 hover:border-orange-300 transition-colors"
                        >
                          <div className="flex items-center gap-3">
                            <div className="w-8 h-8 rounded-lg bg-orange-100 text-orange-700 flex items-center justify-center font-bold text-xs">
                              {item.provider.slice(0, 2).toUpperCase()}
                            </div>
                            <div>
                              <div className="text-xs font-bold text-slate-900">{item.to}</div>
                              <div className="text-[11px] font-mono text-slate-500">
                                Code: <span className="font-bold text-orange-600">{item.code}</span>
                              </div>
                            </div>
                          </div>
                          <div className="text-right">
                            <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-emerald-50 text-emerald-700 border border-emerald-200">
                              0ms
                            </span>
                            <div className="text-[10px] text-slate-400 mt-1">{item.time}</div>
                          </div>
                        </div>
                      ))}
                    </div>

                    {/* Interactive Trigger Controls */}
                    <div className="pt-2 flex flex-wrap gap-2">
                      <button
                        type="button"
                        onClick={() => handleTriggerSimulate("Twilio")}
                        className="flex-1 inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg bg-slate-900 hover:bg-slate-800 text-white text-xs font-semibold shadow-xs transition-all active:scale-[0.98]"
                      >
                        <RefreshCw className={cn("w-3.5 h-3.5", demoPulsing && "animate-spin")} />
                        <span>Simulate Twilio</span>
                      </button>
                      <button
                        type="button"
                        onClick={() => handleTriggerSimulate("Termii")}
                        className="flex-1 inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg bg-white border border-slate-300 hover:bg-slate-50 text-slate-800 text-xs font-semibold shadow-xs transition-colors"
                      >
                        <RefreshCw className={cn("w-3.5 h-3.5", demoPulsing && "animate-spin")} />
                        <span>Simulate Termii</span>
                      </button>
                    </div>
                  </div>

                  {/* Right Column: Hero Focus Card (Captured OTP Extraction) */}
                  <div className="lg:col-span-7">
                    <div className="rounded-2xl bg-white border-2 border-orange-200/90 p-6 sm:p-8 shadow-xl space-y-6 relative overflow-hidden">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-orange-500 to-amber-500 text-white flex items-center justify-center font-bold shadow-md shadow-orange-500/20">
                            <KeyRound className="w-5 h-5" />
                          </div>
                          <div>
                            <div className="text-sm font-bold text-slate-900">
                              Captured Verification Token
                            </div>
                            <div className="text-xs font-mono text-slate-500">
                              Recipient: {recipientNumber}
                            </div>
                          </div>
                        </div>
                        <span className="text-xs font-bold px-2.5 py-1 rounded-full bg-emerald-50 text-emerald-800 border border-emerald-200">
                          Extracted (0ms)
                        </span>
                      </div>

                      {/* Giant Number Tiles */}
                      <div
                        className={cn(
                          "flex items-center justify-center gap-3 sm:gap-4 py-2",
                          demoPulsing && "scale-105 transition-transform"
                        )}
                      >
                        {liveOtp.split("").map((digit, idx) => {
                          const tileKey = `hero-otp-tile-${digit}-${idx}`;
                          return (
                            <div
                              key={tileKey}
                              className="w-14 h-18 sm:w-16 sm:h-20 rounded-2xl bg-slate-950 text-white font-mono font-black text-3xl sm:text-4xl flex items-center justify-center shadow-lg border border-slate-800"
                            >
                              {digit}
                            </div>
                          );
                        })}
                      </div>

                      {/* Delivery Lifecycle Progression */}
                      <div className="space-y-2 pt-2 border-t border-slate-100">
                        <div className="flex items-center justify-between text-xs font-semibold text-slate-600">
                          <span>Delivery Track Progression</span>
                          <span className="text-emerald-600 font-bold">Status: Delivered</span>
                        </div>
                        <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden flex">
                          <div className="bg-orange-500 h-full w-1/3" />
                          <div className="bg-amber-500 h-full w-1/3" />
                          <div className="bg-emerald-500 h-full w-1/3" />
                        </div>
                        <div className="flex justify-between text-[11px] font-mono text-slate-400">
                          <span>Queued (0ms)</span>
                          <span>Sending (12ms)</span>
                          <span>Delivered (48ms)</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Compatibility Ecosystem Ticker */}
          <div className="mt-14 pt-8 border-t border-slate-200/80 text-center space-y-4">
            <p className="text-xs font-bold uppercase tracking-wider text-slate-500">
              Tested and verified with official client libraries and testing frameworks
            </p>
            <div className="flex flex-wrap items-center justify-center gap-8 sm:gap-14 opacity-70 hover:opacity-100 transition-opacity font-bold text-sm tracking-tight text-slate-700">
              <span>TWILIO SDK</span>
              <span>TERMII API</span>
              <span>NODEMAILER</span>
              <span>PLAYWRIGHT</span>
              <span>CYPRESS</span>
              <span>LARAVEL</span>
              <span>DJANGO</span>
            </div>
          </div>
        </div>
      </motion.section>

      {/* NEW INTERACTIVE STORY TELLING SCROLL SECTION */}
      <StoryScroll />

      {/* 3. "MESSAGES THAT STAY LOCAL" (100% Hermetic Gauge & Privacy) */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        className="py-20 lg:py-28 bg-slate-50 border-y border-slate-200"
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
            {/* Left: Dual Radial SVG Gauge */}
            <div className="lg:col-span-4 flex flex-col items-center justify-center text-center">
              <div className="relative w-48 h-48 flex items-center justify-center">
                <svg
                  className="w-full h-full -rotate-90"
                  viewBox="0 0 100 100"
                  role="img"
                  aria-label="100% hermetic local sandbox gauge"
                >
                  <title>100% local hermetic sandbox</title>
                  <circle
                    cx="50"
                    cy="50"
                    r="40"
                    fill="transparent"
                    stroke="#E2E8F0"
                    strokeWidth="8"
                  />
                  <circle
                    cx="50"
                    cy="50"
                    r="40"
                    fill="transparent"
                    stroke="#10B981"
                    strokeWidth="8"
                    strokeDasharray="251.2"
                    strokeDashoffset="0"
                    strokeLinecap="round"
                  />
                </svg>
                <div className="absolute flex flex-col items-center">
                  <span className="text-4xl font-black text-slate-900 tracking-tight">
                    100<span className="text-emerald-500 text-2xl">%</span>
                  </span>
                  <span className="text-[11px] font-bold text-slate-500 uppercase tracking-wider">
                    Hermetic Sandbox
                  </span>
                </div>
              </div>
              <p className="text-xs text-slate-500 mt-3 font-semibold">
                Zero outbound packets leave your workstation
              </p>
            </div>

            {/* Center: Copy */}
            <div className="lg:col-span-5 space-y-4">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-orange-100 text-orange-800 text-xs font-bold uppercase tracking-wider">
                Local Privacy Guaranteed
              </div>
              <h2 className="text-3xl sm:text-4xl font-extrabold tracking-tight text-slate-950">
                Messages that <br />
                stay on your machine.
              </h2>
              <p className="text-sm sm:text-base text-slate-600 leading-relaxed">
                Never accidentally blast test messages to real customer handsets, drain live carrier
                budgets in staging, or wait for external carrier network latency during automated CI
                runs.
              </p>
            </div>

            {/* Right: Metrics Card */}
            <div className="lg:col-span-3">
              <div className="rounded-2xl bg-white border border-slate-200 p-6 shadow-sm space-y-4">
                <div className="text-[11px] font-bold uppercase tracking-wider text-orange-600">
                  Local Sandbox Telemetry
                </div>
                <div className="grid grid-cols-3 gap-2 text-center pt-1 border-t border-slate-100">
                  <div>
                    <div className="text-2xl font-black text-slate-900 font-mono">0ms</div>
                    <div className="text-[10px] text-slate-500 uppercase font-semibold">
                      Latency
                    </div>
                  </div>
                  <div className="border-x border-slate-100">
                    <div className="text-2xl font-black text-emerald-600 font-mono">$0</div>
                    <div className="text-[10px] text-slate-500 uppercase font-semibold">
                      Forever
                    </div>
                  </div>
                  <div>
                    <div className="text-2xl font-black text-slate-900 font-mono">100%</div>
                    <div className="text-[10px] text-slate-500 uppercase font-semibold">
                      Offline
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </motion.section>

      {/* 4. THE ICONIC CURVED ORANGE STATEMENT QUOTE BANNER */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        className="py-14 bg-white"
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="rounded-3xl bg-gradient-to-tr from-orange-600 via-orange-500 to-amber-500 text-white p-8 sm:p-14 lg:p-16 shadow-2xl relative overflow-hidden">
            {/* Transparent decorative quotation mark in top right */}
            <div className="absolute right-8 top-4 text-white/15 text-8xl sm:text-9xl font-serif select-none pointer-events-none">
              â€œ
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center relative z-10">
              <div className="lg:col-span-7">
                <h2 className="text-3xl sm:text-4xl lg:text-5xl font-black tracking-tight leading-tight">
                  Every message tested locally protects production integrity.
                </h2>
              </div>
              <div className="lg:col-span-5 text-orange-50 text-sm sm:text-base leading-relaxed space-y-4">
                <p>
                  We believe testing authentication flows, webhooks, and multi-part SMS
                  shouldn&apos;t require external cloud accounts, credit cards, or carrier rate
                  limits.
                </p>
                <p>
                  mocksms runs as a single, self-contained Go binary with embedded SQLite and a live
                  web console, delivering total fidelity and zero flakiness.
                </p>
              </div>
            </div>
          </div>
        </div>
      </motion.section>

      {/* 5. "EVERY TOOL YOUR TEAM NEEDS" (Interactive Sandbox Composer & Live Segment Counter) */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        id="composer"
        className="py-20 lg:py-28 bg-white"
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-12">
          {/* Header Row */}
          <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 pb-4">
            <div className="max-w-xl space-y-3">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-orange-100 text-orange-800 text-xs font-bold uppercase tracking-wider">
                Interactive Playground
              </div>
              <h2 className="text-3xl sm:text-4xl font-black tracking-tight text-slate-950">
                Every tool your <br />
                team needs.
              </h2>
            </div>
            <p className="text-sm text-slate-500 max-w-sm">
              Try the live composer below to see automatic character segmentation and encoding
              detection in action.
            </p>
          </div>

          {/* Interactive Feature Showcase Grid */}
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
            {/* Left: Interactive Live SMS Composer Card */}
            <div className="lg:col-span-7">
              <div className="rounded-2xl bg-white border border-slate-200 shadow-xl p-6 sm:p-7 space-y-5">
                {/* Tabs */}
                <div className="flex items-center gap-6 border-b border-slate-100 pb-3 text-xs font-bold">
                  <button
                    type="button"
                    onClick={() => setComposerMode("sms")}
                    className={cn(
                      "pb-2 transition-colors",
                      composerMode === "sms"
                        ? "text-orange-600 border-b-2 border-orange-500"
                        : "text-slate-500 hover:text-slate-900"
                    )}
                  >
                    Quick SMS
                  </button>
                  <button
                    type="button"
                    onClick={() => setComposerMode("smtp")}
                    className={cn(
                      "pb-2 transition-colors",
                      composerMode === "smtp"
                        ? "text-orange-600 border-b-2 border-orange-500"
                        : "text-slate-500 hover:text-slate-900"
                    )}
                  >
                    SMTP Email
                  </button>
                  <button
                    type="button"
                    onClick={() => setComposerMode("inbound")}
                    className={cn(
                      "pb-2 transition-colors",
                      composerMode === "inbound"
                        ? "text-orange-600 border-b-2 border-orange-500"
                        : "text-slate-500 hover:text-slate-900"
                    )}
                  >
                    Simulate Inbound
                  </button>
                </div>

                {/* Sender ID & Recipients */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <span className="text-xs font-bold text-slate-700 block mb-1">Sender ID</span>
                    <div className="p-2.5 rounded-lg border border-slate-200 bg-slate-50 text-xs font-bold text-slate-900 flex items-center justify-between">
                      <span>MOCKSMS (Sandbox Approved)</span>
                      <Check className="w-3.5 h-3.5 text-emerald-600" />
                    </div>
                  </div>
                  <div>
                    <span className="text-xs font-bold text-slate-700 block mb-1">
                      Recipient (E.164)
                    </span>
                    <div className="p-2.5 rounded-lg border border-slate-200 bg-white text-xs font-mono text-slate-800">
                      +233 24 123 4567
                    </div>
                  </div>
                </div>

                {/* Live Editable Message Body */}
                <div>
                  <div className="flex items-center justify-between mb-1.5">
                    <label
                      htmlFor={composerTextId}
                      className="text-xs font-bold text-slate-700 cursor-pointer"
                    >
                      Message Payload (Type to calculate segments)
                    </label>
                    <div className="flex items-center gap-2">
                      <span
                        className={cn(
                          "text-[10px] font-bold px-2 py-0.5 rounded",
                          isUnicode ? "bg-amber-100 text-amber-900" : "bg-slate-100 text-slate-700"
                        )}
                      >
                        {isUnicode ? "Unicode (UCS-2)" : "GSM-7 Standard"}
                      </span>
                      <span className="text-xs font-mono font-bold text-orange-600">
                        {composerText.length} chars Â· {segmentCount} Segment
                        {segmentCount > 1 ? "s" : ""}
                      </span>
                    </div>
                  </div>

                  <textarea
                    id={composerTextId}
                    rows={3}
                    value={composerText}
                    onChange={(e) => setComposerText(e.target.value)}
                    className="w-full p-3 rounded-lg border border-slate-300 focus:border-orange-500 focus:ring-1 focus:ring-orange-500 text-xs text-slate-800 leading-relaxed font-sans outline-hidden resize-none transition-all"
                  />
                </div>

                {/* Send action */}
                <div className="flex items-center justify-between pt-1">
                  <div className="text-[11px] text-slate-500 flex items-center gap-1.5">
                    <ShieldCheck className="w-3.5 h-3.5 text-emerald-600" />
                    <span>Hermetic: Dispatches directly to embedded SQLite</span>
                  </div>
                  <Link
                    to="/inbox"
                    className="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-orange-500 hover:bg-orange-600 text-white font-bold text-xs shadow-md transition-all active:scale-[0.98]"
                  >
                    <Send className="w-3.5 h-3.5" />
                    <span>Send to Sandbox</span>
                  </Link>
                </div>
              </div>
            </div>

            {/* Right: Feature Highlights */}
            <div className="lg:col-span-5 space-y-6">
              <div className="w-12 h-12 rounded-xl bg-orange-100 text-orange-600 flex items-center justify-center shadow-inner">
                <Cpu className="w-6 h-6" />
              </div>

              <h3 className="text-2xl font-bold text-slate-950">
                Segment Counting &amp; Validation
              </h3>

              <p className="text-sm text-slate-600 leading-relaxed">
                Compute exact GSM-7 vs. Unicode UCS-2 character boundaries before going to
                production. Verify multi-part concatenated SMS segmentation and avoid unexpected
                carrier invoice surprises.
              </p>

              <ul className="space-y-3 text-xs text-slate-700">
                <li className="flex items-center gap-2.5">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>Automatic GSM-7 160-char and UCS-2 70-char segment boundary parsing</span>
                </li>
                <li className="flex items-center gap-2.5">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>Automatic 4-8 digit OTP code and magic verification link detection</span>
                </li>
                <li className="flex items-center gap-2.5">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>Inbound STOP and START keyword opt-out suppression list compliance</span>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </motion.section>

      {/* 6. "SMARTER TESTING, BUILT IN" (Magic Numbers & CI Endpoints) */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        className="py-20 lg:py-28 bg-slate-50 border-t border-slate-200"
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-12">
          {/* Header Row */}
          <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
            <div className="max-w-xl space-y-2">
              <h2 className="text-3xl sm:text-4xl font-black tracking-tight text-slate-950">
                Smarter testing, <br />
                built in.
              </h2>
            </div>
            <p className="text-sm text-slate-500 max-w-sm">
              Engineered specifically to solve the headaches of mocking third-party communication
              APIs.
            </p>
          </div>

          {/* 3 Interactive Feature Cards */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {/* Card 1: Interactive Magic Pattern Numbers */}
            <div className="rounded-2xl bg-white border border-slate-200 p-6 shadow-sm space-y-4 hover:shadow-md transition-shadow">
              <div className="space-y-2">
                <div className="text-[11px] font-bold text-slate-500 uppercase">
                  Select Magic Number Edge Case
                </div>
                <div className="grid grid-cols-2 gap-1">
                  {magicNumberCases.map((m, idx) => (
                    <button
                      key={m.number}
                      type="button"
                      onClick={() => setSelectedMagicIndex(idx)}
                      className={cn(
                        "p-1.5 rounded text-[11px] font-mono text-left transition-colors",
                        selectedMagicIndex === idx
                          ? "bg-slate-900 text-white font-bold"
                          : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                      )}
                    >
                      {m.number.slice(-6)}
                    </button>
                  ))}
                </div>

                <div className="p-3 bg-rose-50 border border-rose-200 rounded-lg text-xs font-mono space-y-1">
                  <div className="text-rose-900 font-bold">{currentMagic.label}</div>
                  <div className="text-[11px] text-rose-700">{currentMagic.simulatedError}</div>
                </div>
              </div>

              <h3 className="text-lg font-bold text-slate-950">Magic Pattern Numbers</h3>
              <p className="text-xs text-slate-600 leading-relaxed">
                Trigger edge cases like undelivered messages, carrier blacklists, and rate limits
                instantly without complex mocking frameworks.
              </p>
            </div>

            {/* Card 2: Deterministic Sleep-Free CI */}
            <div className="rounded-2xl bg-white border border-slate-200 p-6 shadow-sm space-y-4 hover:shadow-md transition-shadow">
              <div className="p-4 rounded-xl bg-slate-900 text-white font-mono text-xs space-y-2">
                <div className="text-[11px] text-slate-400 font-bold uppercase">
                  Sleep-Free CI Endpoint
                </div>
                <div className="text-emerald-400">GET /messages/wait?since=...</div>
                <div className="text-slate-300 text-[11px]">
                  &gt; Resolves instantly when message lands. Zero polling delays.
                </div>
              </div>

              <h3 className="text-lg font-bold text-slate-950">Sleep-Free CI Endpoints</h3>
              <p className="text-xs text-slate-600 leading-relaxed">
                Eliminate flaky time.Sleep() in automated test runs. Playwright and Cypress tests
                resume the millisecond a message is received.
              </p>
            </div>

            {/* Card 3: Exact Provider Fidelity */}
            <div className="rounded-2xl bg-white border border-slate-200 p-6 shadow-sm space-y-4 hover:shadow-md transition-shadow">
              <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-2 font-mono text-xs">
                <div className="text-[11px] font-bold text-slate-500 uppercase">
                  Adapter Translation
                </div>
                <div className="flex justify-between p-1.5 bg-white rounded border border-slate-200">
                  <span>Twilio SID</span>
                  <span className="text-orange-600 font-bold">SM_mock...</span>
                </div>
                <div className="flex justify-between p-1.5 bg-white rounded border border-slate-200">
                  <span>Termii ID</span>
                  <span className="text-orange-600 font-bold">msg_id_01...</span>
                </div>
              </div>

              <h3 className="text-lg font-bold text-slate-950">Exact Provider Fidelity</h3>
              <p className="text-xs text-slate-600 leading-relaxed">
                Returns identical HTTP status codes, error structures, SID formats, and status
                callbacks matching real production endpoints.
              </p>
            </div>
          </div>
        </div>
      </motion.section>

      {/* 7. "RICH ON FEATURES, CI/CD READY" (Bento Grid of 6 Cards) */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        id="features"
        className="py-20 lg:py-28 bg-white"
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-12">
          {/* Header Row */}
          <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
            <div className="max-w-xl space-y-2">
              <h2 className="text-3xl sm:text-4xl font-black tracking-tight text-slate-950">
                Rich on features, <br />
                CI/CD ready.
              </h2>
            </div>
            <p className="text-sm text-slate-500 max-w-sm">
              From solitary developer laptops to automated GitHub Actions runners, mocksms handles
              it all.
            </p>
          </div>

          {/* 6-Card Bento Grid */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Card 1: Live Event Streams */}
            <div className="rounded-2xl bg-amber-50/50 border border-orange-200/80 p-6 space-y-4 shadow-sm hover:-translate-y-1 hover:shadow-md transition-all duration-300">
              <div className="space-y-2">
                <div className="text-xs font-bold text-slate-700">Real-Time Event Stream</div>
                <div className="space-y-1 font-mono text-[11px] text-slate-600">
                  <div className="p-2 bg-white rounded border border-orange-200 flex justify-between">
                    <span>event: message.created</span>
                    <span className="text-emerald-600 font-bold">SSE Live</span>
                  </div>
                </div>
              </div>
              <h3 className="text-base font-bold text-slate-950">Live Server-Sent Events</h3>
              <p className="text-xs text-slate-600">
                Instant UI inbox updates via SSE with non-blocking copy-on-write event dispatch.
              </p>
            </div>

            {/* Card 2: Webhook Replay & Retries */}
            <div className="rounded-2xl bg-white border border-slate-200 p-6 space-y-4 shadow-sm hover:-translate-y-1 hover:shadow-md transition-all duration-300">
              <div className="space-y-1.5 p-3 rounded-lg bg-slate-50 border border-slate-200 text-xs font-mono">
                <div className="flex justify-between">
                  <span>POST /webhooks/sms</span>
                  <span className="text-emerald-600 font-bold">200 OK</span>
                </div>
                <div className="text-[10px] text-slate-400">HMAC-SHA1 Signature Verified</div>
              </div>
              <h3 className="text-base font-bold text-slate-950">Webhook Testing &amp; Replay</h3>
              <p className="text-xs text-slate-600">
                Test your backend status callbacks with one-click webhook replays and payload logs.
              </p>
            </div>

            {/* Card 3: Multi-Project Scoping */}
            <div className="rounded-2xl bg-white border border-slate-200 p-6 space-y-4 shadow-sm hover:-translate-y-1 hover:shadow-md transition-all duration-300">
              <div className="flex items-center gap-2 p-2.5 rounded-lg bg-slate-50 border border-slate-200 text-xs">
                <span className="font-semibold text-slate-800">Project: default</span>
                <span className="ml-auto text-[10px] px-2 py-0.5 rounded bg-emerald-100 text-emerald-800 font-bold">
                  Isolated SQLite
                </span>
              </div>
              <h3 className="text-base font-bold text-slate-950">
                Project Scoping &amp; Isolation
              </h3>
              <p className="text-xs text-slate-600">
                Isolate test suites and developers using dedicated project headers or query
                parameters.
              </p>
            </div>

            {/* Card 4: Email Sandboxing */}
            <div className="rounded-2xl bg-white border border-slate-200 p-6 space-y-4 shadow-sm hover:-translate-y-1 hover:shadow-md transition-all duration-300">
              <div className="p-3 bg-slate-50 rounded-lg border border-slate-200 font-mono text-xs flex justify-between">
                <span>SMTP Port :1025</span>
                <span className="text-emerald-600 font-bold">Listening</span>
              </div>
              <h3 className="text-base font-bold text-slate-950">Sandboxed Email SMTP</h3>
              <p className="text-xs text-slate-600">
                Catch transactional emails with sandboxed HTML previews, raw MIME viewing, and
                attachment downloads.
              </p>
            </div>

            {/* Card 5: Partial Batch Rejection */}
            <div className="rounded-2xl bg-amber-50/50 border border-orange-200/80 p-6 space-y-4 shadow-sm hover:-translate-y-1 hover:shadow-md transition-all duration-300">
              <div className="flex items-center justify-between text-xs font-bold text-slate-700 py-1.5 font-mono">
                <span>Batch: 500 Accepted</span>
                <span className="text-rose-600">2 Rejected</span>
              </div>
              <h3 className="text-base font-bold text-slate-950">Partial Batch Validation</h3>
              <p className="text-xs text-slate-600">
                Transactional bulk insertion that accepts valid items while returning structured
                rejection lists.
              </p>
            </div>

            {/* Card 6: Zero-Dependency Single Binary (DARK CARD) */}
            <div className="rounded-2xl bg-slate-950 text-white p-6 space-y-4 border-t-4 border-t-orange-500 shadow-xl hover:-translate-y-1 hover:shadow-2xl transition-all duration-300">
              <div className="flex items-center justify-between text-xs">
                <span className="font-mono text-orange-400 font-bold">SINGLE GO BINARY</span>
                <HardDrive className="w-4 h-4 text-orange-400" />
              </div>
              <h3 className="text-base font-bold text-white">Zero External Dependencies</h3>
              <p className="text-xs text-slate-400">
                No Docker required, no PostgreSQL, no Redis. Pure Go binary with embedded SQLite and
                embedded React console.
              </p>
              <div className="pt-1">
                <div className="w-full py-1.5 text-center rounded bg-orange-500 text-white font-bold text-[11px]">
                  ~24MB Static Executable
                </div>
              </div>
            </div>
          </div>
        </div>
      </motion.section>

      {/* 8. "BUILD WITH THE MOCKSMS API" (Interactive Code Terminal & Orange Dotted Grid) */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        id="api"
        className="relative py-20 lg:py-28 bg-[#FFF8F0] overflow-hidden border-t border-orange-100"
      >
        {/* Subtle Orange Polka-Dot Grid Pattern */}
        <div
          className="absolute inset-0 opacity-25 pointer-events-none"
          style={{
            backgroundImage: "radial-gradient(circle at 2px 2px, #f97316 1.5px, transparent 0)",
            backgroundSize: "24px 24px",
          }}
        />

        <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
            {/* Left: API Steps */}
            <div className="lg:col-span-5 space-y-6">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-orange-100 text-orange-800 text-xs font-mono font-bold">
                <Code2 className="w-3.5 h-3.5" />
                <span>Zero SDK Changes Required</span>
              </div>

              <h2 className="text-3xl sm:text-4xl font-black tracking-tight text-slate-950">
                Build with the mocksms API
              </h2>

              <p className="text-sm sm:text-base text-slate-600 leading-relaxed">
                Two lines of code to point your existing provider libraries to mocksms. No special
                SDK forks or complex mocks needed.
              </p>

              <div className="space-y-4 text-xs text-slate-700">
                <div className="flex items-start gap-3">
                  <div className="w-6 h-6 rounded-full bg-orange-500 text-white flex items-center justify-center font-bold shrink-0 mt-0.5">
                    1
                  </div>
                  <div>
                    <span className="font-bold text-slate-900 block">Point Base URL</span>
                    <span>Set client API endpoint to http://127.0.0.1:4010.</span>
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <div className="w-6 h-6 rounded-full bg-orange-500 text-white flex items-center justify-center font-bold shrink-0 mt-0.5">
                    2
                  </div>
                  <div>
                    <span className="font-bold text-slate-900 block">Use Dummy Credentials</span>
                    <span>Any non-empty Account SID and Auth Token work in sandbox mode.</span>
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <div className="w-6 h-6 rounded-full bg-orange-500 text-white flex items-center justify-center font-bold shrink-0 mt-0.5">
                    3
                  </div>
                  <div>
                    <span className="font-bold text-slate-900 block">Inspect &amp; Assert</span>
                    <span>
                      Verify delivery in the local web console or query via /messages/wait.
                    </span>
                  </div>
                </div>
              </div>

              <Link
                to="/inbox"
                className="inline-flex items-center justify-center gap-2 px-6 py-3 rounded-full bg-orange-500 hover:bg-orange-600 text-white font-bold text-xs shadow-md transition-all active:scale-[0.98]"
              >
                <span>Open Web Inbox</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </Link>
            </div>

            {/* Right: Dark macOS Code Terminal */}
            <div className="lg:col-span-7">
              <div className="rounded-2xl bg-slate-950 border border-slate-800 shadow-2xl overflow-hidden">
                {/* Window Chrome & Tabs */}
                <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 bg-slate-900 border-b border-slate-800">
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 rounded-full bg-red-500" />
                    <div className="w-3 h-3 rounded-full bg-yellow-500" />
                    <div className="w-3 h-3 rounded-full bg-emerald-500" />
                    <span className="text-xs font-mono text-slate-400 ml-2">
                      {currentSnippet.filename}
                    </span>
                  </div>

                  {/* Tabs */}
                  <div className="flex items-center gap-1 bg-slate-950 p-0.5 rounded-lg text-xs font-mono">
                    {codeSnippets.map((snippet) => (
                      <button
                        key={snippet.id}
                        type="button"
                        onClick={() => setActiveCodeSnippet(snippet.id)}
                        className={cn(
                          "px-2.5 py-1 rounded transition-colors",
                          activeCodeSnippet === snippet.id
                            ? "bg-slate-800 text-orange-400 font-bold"
                            : "text-slate-400 hover:text-slate-200"
                        )}
                      >
                        {snippet.label}
                      </button>
                    ))}
                  </div>

                  <button
                    type="button"
                    onClick={handleCopySnippet}
                    className="p-1.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs flex items-center gap-1 transition-colors"
                    aria-label="Copy code snippet"
                  >
                    {copiedSnippet ? (
                      <Check className="w-3.5 h-3.5 text-emerald-400" />
                    ) : (
                      <Copy className="w-3.5 h-3.5" />
                    )}
                  </button>
                </div>

                {/* Code Area */}
                <div className="p-5 font-mono text-xs text-slate-200 leading-relaxed overflow-x-auto">
                  <pre>{currentSnippet.code}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </motion.section>

      {/* 9. "FREE FOR DEV. PREDICTABLE FOR PROD." SECTION (Pricing & Cost Savings) */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        id="savings"
        className="py-20 lg:py-28 bg-white"
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-12">
          {/* Header & Pill Switcher */}
          <div className="text-center max-w-2xl mx-auto space-y-4">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-orange-100 text-orange-800 text-xs font-bold uppercase tracking-wider">
              Transparent Economics
            </div>
            <h2 className="text-3xl sm:text-4xl font-black tracking-tight text-slate-950">
              Free for dev. Predictable for prod.
            </h2>
            <p className="text-sm text-slate-600">
              mocksms is 100% free and open-source. Calculate how much you save during development
              and plan your production go-live budget.
            </p>

            {/* Switcher Pills */}
            <div className="inline-flex items-center p-1 bg-slate-100 rounded-full text-xs font-bold">
              <button
                type="button"
                onClick={() => setPricingInterval("dev")}
                className={cn(
                  "px-4 py-1.5 rounded-full transition-colors",
                  pricingInterval === "dev"
                    ? "bg-slate-900 text-white shadow-xs"
                    : "text-slate-600 hover:text-slate-900"
                )}
              >
                Local Dev
              </button>
              <button
                type="button"
                onClick={() => setPricingInterval("ci")}
                className={cn(
                  "px-4 py-1.5 rounded-full transition-colors",
                  pricingInterval === "ci"
                    ? "bg-slate-900 text-white shadow-xs"
                    : "text-slate-600 hover:text-slate-900"
                )}
              >
                CI Automation
              </button>
              <button
                type="button"
                onClick={() => setPricingInterval("prod")}
                className={cn(
                  "px-4 py-1.5 rounded-full transition-colors",
                  pricingInterval === "prod"
                    ? "bg-slate-900 text-white shadow-xs"
                    : "text-slate-600 hover:text-slate-900"
                )}
              >
                Production Go-Live
              </button>
            </div>
          </div>

          {/* 3 Pricing Cards */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8 items-center pt-4">
            {/* Card 1: Starter / Local Dev */}
            <div className="rounded-2xl bg-white border border-slate-200 p-6 shadow-sm space-y-4">
              <div className="text-xs font-bold uppercase text-slate-500">Local Workstations</div>
              <div className="text-3xl font-extrabold text-slate-900 font-mono">
                $0 <span className="text-xs font-normal text-slate-500">forever</span>
              </div>
              <p className="text-xs text-slate-600">
                For developers building auth flows on laptops.
              </p>
              <ul className="space-y-2 text-xs text-slate-700 pt-3 border-t border-slate-100">
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>Unlimited local SMS &amp; emails</span>
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>Embedded SQLite storage &amp; web UI</span>
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>Single self-contained Go binary</span>
                </li>
              </ul>
              <div className="pt-2">
                <Link
                  to="/inbox"
                  className="w-full inline-flex justify-center py-2.5 rounded-lg border border-slate-300 hover:bg-slate-50 text-slate-800 font-bold text-xs transition-colors"
                >
                  Start Developing Free
                </Link>
              </div>
            </div>

            {/* Card 2: CI Pipelines (Highlighted Solid Vibrant Orange Card) */}
            <div className="rounded-2xl bg-gradient-to-tr from-orange-600 to-amber-500 text-white p-7 shadow-xl space-y-4 relative md:-mt-4 md:mb-[-1rem] flex flex-col justify-center">
              <div className="absolute -top-3 right-6 px-3 py-0.5 rounded-full bg-slate-950 text-white text-[10px] font-bold uppercase tracking-wider">
                Automated Testing
              </div>
              <div className="text-xs font-bold uppercase text-orange-100">CI/CD Pipelines</div>
              <div className="text-4xl font-black text-white font-mono">
                $0 <span className="text-xs font-normal text-orange-200">in test suites</span>
              </div>
              <p className="text-xs text-orange-50">
                For GitHub Actions, Playwright, and staging environments.
              </p>
              <ul className="space-y-2 text-xs text-white pt-3 border-t border-orange-400/60">
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-white shrink-0" />
                  <span>Distroless lightweight Docker image</span>
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-white shrink-0" />
                  <span>Deterministic /messages/wait endpoints</span>
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-white shrink-0" />
                  <span>Zero carrier rate limits or bill spikes</span>
                </li>
              </ul>
              <div className="pt-2">
                <Link
                  to="/inbox"
                  className="w-full inline-flex justify-center py-2.5 rounded-lg bg-white hover:bg-orange-50 text-orange-600 font-bold text-xs shadow-md transition-colors"
                >
                  Run in CI Free
                </Link>
              </div>
            </div>

            {/* Card 3: Production Go-Live */}
            <div className="rounded-2xl bg-white border border-slate-200 p-6 shadow-sm space-y-4">
              <div className="text-xs font-bold uppercase text-slate-500">Production Launch</div>
              <div className="text-3xl font-extrabold text-slate-900 font-mono">
                Go-Live <span className="text-xs font-normal text-slate-500">with real telcos</span>
              </div>
              <p className="text-xs text-slate-600">
                Switch endpoints back to real Twilio or Termii.
              </p>
              <ul className="space-y-2 text-xs text-slate-700 pt-3 border-t border-slate-100">
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>Zero code refactoring needed</span>
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>Same SDK method calls and signatures</span>
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>Deliver to real customer handsets</span>
                </li>
              </ul>
              <div className="pt-2">
                <a
                  href="#api"
                  className="w-full inline-flex justify-center py-2.5 rounded-lg border border-slate-300 hover:bg-slate-50 text-slate-800 font-bold text-xs transition-colors"
                >
                  View Deployment Guide
                </a>
              </div>
            </div>
          </div>

          {/* Volume Savings Calculator */}
          <div className="max-w-xl mx-auto p-5 rounded-2xl bg-slate-50 border border-slate-200 space-y-3">
            <div className="flex items-center justify-between text-xs font-bold text-slate-700">
              <label htmlFor={volumeSliderId} className="cursor-pointer">
                Simulate Monthly Test Volume:
              </label>
              <span className="font-mono text-sm font-bold text-slate-900">
                {volume.toLocaleString()} test messages
              </span>
            </div>
            <input
              id={volumeSliderId}
              type="range"
              min="5000"
              max="200000"
              step="5000"
              value={volume}
              onChange={(e) => setVolume(Number(e.target.value))}
              className="w-full h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer accent-orange-500"
            />
            <div className="flex items-center justify-between text-xs text-slate-500 pt-1 font-semibold">
              <span>Saved vs. Twilio: ${twilioSaved} / mo</span>
              <span className="font-bold text-orange-600">
                Saved vs. Termii: ${termiiSaved} / mo
              </span>
            </div>
          </div>
        </div>
      </motion.section>

      {/* 10. "THERE'S MORE TO MOCKSMS" SECTION */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        className="py-20 lg:py-28 bg-slate-50 border-t border-slate-200"
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-12">
          {/* Header Row */}
          <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
            <div className="max-w-xl space-y-2">
              <h2 className="text-3xl sm:text-4xl font-black tracking-tight text-slate-950">
                There&apos;s more <br />
                to mocksms
              </h2>
            </div>
            <p className="text-sm text-slate-500 max-w-sm">
              Engineered with full fidelity to make local sandbox testing feel identical to
              production.
            </p>
          </div>

          {/* 2 Large Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            {/* Left Card: Webhooks & Live Logs */}
            <div className="rounded-3xl bg-white border border-slate-200 p-8 shadow-sm space-y-6 flex flex-col justify-between">
              <div className="space-y-4">
                <div className="w-10 h-10 rounded-xl bg-orange-100 text-orange-600 flex items-center justify-center">
                  <Terminal className="w-5 h-5" />
                </div>
                <h3 className="text-2xl font-bold text-slate-950">
                  Exact Webhook Signatures &amp; Status Transitions
                </h3>
                <p className="text-sm text-slate-600 leading-relaxed">
                  Simulate full message lifecycles from queued to sending, sent, and delivered with
                  realistic backoff and Twilio HMAC-SHA1 webhook signatures.
                </p>
                <div>
                  <Link
                    to="/inbox"
                    className="inline-flex items-center gap-1.5 text-xs font-bold text-orange-600 hover:text-orange-700"
                  >
                    <span>Inspect live webhooks</span>
                    <ArrowRight className="w-3.5 h-3.5" />
                  </Link>
                </div>
              </div>

              {/* Graphic Mockup */}
              <div className="p-4 rounded-xl bg-slate-950 text-slate-300 font-mono text-xs overflow-x-auto shadow-inner">
                <pre>{`{
  "event": "message.delivered",
  "id": "msg_01J98X7...",
  "to": "+233241234567",
  "provider": "termii",
  "status": "delivered",
  "segments": 1
}`}</pre>
              </div>
            </div>

            {/* Right Card: Automatic Data Retention & Privacy Pruning */}
            <div className="rounded-3xl bg-white border border-slate-200 p-8 shadow-sm space-y-6 flex flex-col justify-between">
              <div className="space-y-4">
                <div className="w-10 h-10 rounded-xl bg-orange-100 text-orange-600 flex items-center justify-center">
                  <ShieldCheck className="w-5 h-5" />
                </div>
                <h3 className="text-2xl font-bold text-slate-950">
                  Automatic Data Retention &amp; Privacy Pruning
                </h3>
                <p className="text-sm text-slate-600 leading-relaxed">
                  Configurable message retention TTL, automated cascading SQLite cleanup, and
                  one-click inbox clearing to keep your local drive tidy.
                </p>
                <div>
                  <Link
                    to="/inbox"
                    className="inline-flex items-center gap-1.5 text-xs font-bold text-orange-600 hover:text-orange-700"
                  >
                    <span>Configure retention settings</span>
                    <ArrowRight className="w-3.5 h-3.5" />
                  </Link>
                </div>
              </div>

              {/* Graphic Mockup */}
              <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-2 text-xs">
                <div className="flex items-center justify-between p-2 bg-white rounded border border-slate-200">
                  <span className="font-semibold text-slate-800">Retention Cleanup Job</span>
                  <span className="text-[10px] font-bold bg-emerald-100 text-emerald-800 px-2 py-0.5 rounded">
                    Active (Every 10m)
                  </span>
                </div>
                <div className="flex items-center justify-between p-2 bg-white rounded border border-slate-200">
                  <span className="font-semibold text-slate-800">TTL Expiration Policy</span>
                  <span className="text-[10px] font-bold bg-orange-100 text-orange-800 px-2 py-0.5 rounded">
                    24 Hours Default
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </motion.section>

      {/* 11. CLOSING STATEMENT BANNER */}
      <motion.section
        initial={{ opacity: 0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-100px" }}
        transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
        className="py-16 sm:py-20 bg-gradient-to-r from-orange-600 to-amber-500 text-white"
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
            <div className="lg:col-span-7">
              <h2 className="text-3xl sm:text-4xl lg:text-5xl font-black tracking-tight leading-tight">
                Send your first test <br />
                message in seconds.
              </h2>
            </div>
            <div className="lg:col-span-5 space-y-6">
              <p className="text-orange-50 text-sm sm:text-base leading-relaxed">
                Download the single static binary or run the Docker image. Zero cloud accounts, zero
                API keys, and zero credit card required.
              </p>
              <div className="flex flex-wrap items-center gap-4">
                <Link
                  to="/inbox"
                  className="inline-flex items-center justify-center px-7 py-3.5 rounded-full bg-white hover:bg-orange-50 text-orange-600 font-bold text-sm shadow-md transition-all active:scale-[0.98]"
                >
                  Launch Local Inbox
                </Link>
                <a
                  href="https://github.com/Aeomar999/CommPit"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center justify-center px-7 py-3.5 rounded-full border border-white/40 hover:bg-white/10 text-white font-medium text-sm transition-colors"
                >
                  View on GitHub
                </a>
              </div>
            </div>
          </div>
        </div>
      </motion.section>

      {/* 12. MULTI-COLUMN FOOTER */}
      <footer className="py-16 bg-white border-t border-slate-200 text-xs text-slate-500">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-12">
          <div className="grid grid-cols-1 md:grid-cols-5 gap-8">
            {/* Brand Column */}
            <div className="space-y-4 md:col-span-1">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-orange-500 flex items-center justify-center text-white shadow-sm">
                  <MessageSquare className="w-4 h-4 fill-current" />
                </div>
                <span className="text-lg font-black text-slate-900">
                  mock<span className="text-orange-500">sms</span>
                </span>
              </div>
              <p className="text-slate-500 leading-relaxed">
                The local-first sandbox messaging provider for SMS, OTP and email testing.
              </p>
            </div>

            {/* Column 1: Features */}
            <div className="space-y-3">
              <div className="font-bold uppercase tracking-wider text-slate-900">Features</div>
              <ul className="space-y-2">
                <li>
                  <Link to="/inbox" className="hover:text-slate-900 transition-colors">
                    SMS Sandbox
                  </Link>
                </li>
                <li>
                  <Link to="/inbox" className="hover:text-slate-900 transition-colors">
                    OTP Extraction
                  </Link>
                </li>
                <li>
                  <Link to="/inbox" className="hover:text-slate-900 transition-colors">
                    Email SMTP (:1025)
                  </Link>
                </li>
                <li>
                  <Link to="/inbox" className="hover:text-slate-900 transition-colors">
                    Webhook Replay
                  </Link>
                </li>
                <li>
                  <Link to="/inbox" className="hover:text-slate-900 transition-colors">
                    Magic Test Numbers
                  </Link>
                </li>
              </ul>
            </div>

            {/* Column 2: Adapters */}
            <div className="space-y-3">
              <div className="font-bold uppercase tracking-wider text-slate-900">Adapters</div>
              <ul className="space-y-2">
                <li>
                  <a href="#api" className="hover:text-slate-900 transition-colors">
                    Twilio SMS &amp; Verify
                  </a>
                </li>
                <li>
                  <a href="#api" className="hover:text-slate-900 transition-colors">
                    Termii API
                  </a>
                </li>
                <li>
                  <a href="#api" className="hover:text-slate-900 transition-colors">
                    Local SMTP Server
                  </a>
                </li>
                <li>
                  <Link to="/inbox" className="hover:text-slate-900 transition-colors">
                    Native REST API
                  </Link>
                </li>
                <li>
                  <Link to="/inbox" className="hover:text-slate-900 transition-colors">
                    Server-Sent Events
                  </Link>
                </li>
              </ul>
            </div>

            {/* Column 3: Documentation */}
            <div className="space-y-3">
              <div className="font-bold uppercase tracking-wider text-slate-900">Documentation</div>
              <ul className="space-y-2">
                <li>
                  <a href="#api" className="hover:text-slate-900 transition-colors">
                    Quick Start
                  </a>
                </li>
                <li>
                  <a href="#api" className="hover:text-slate-900 transition-colors">
                    Docker Guide
                  </a>
                </li>
                <li>
                  <a href="#api" className="hover:text-slate-900 transition-colors">
                    CI/CD Integration
                  </a>
                </li>
                <li>
                  <a href="#api" className="hover:text-slate-900 transition-colors">
                    API Quick Reference
                  </a>
                </li>
                <li>
                  <a href="#api" className="hover:text-slate-900 transition-colors">
                    CLI Commands
                  </a>
                </li>
              </ul>
            </div>

            {/* Column 4: Open Source */}
            <div className="space-y-3">
              <div className="font-bold uppercase tracking-wider text-slate-900">Open Source</div>
              <ul className="space-y-2">
                <li>
                  <a
                    href="https://github.com/Aeomar999/CommPit"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:text-slate-900 transition-colors"
                  >
                    GitHub Repository
                  </a>
                </li>
                <li>
                  <a href="#license" className="hover:text-slate-900 transition-colors">
                    MIT License
                  </a>
                </li>
                <li>
                  <a
                    href="https://github.com/Aeomar999/CommPit/issues"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:text-slate-900 transition-colors"
                  >
                    Issue Tracker
                  </a>
                </li>
                <li>
                  <Link to="/inbox" className="hover:text-slate-900 transition-colors">
                    Release Notes (v0.1.0)
                  </Link>
                </li>
              </ul>
            </div>
          </div>

          {/* Bottom Copyright */}
          <div className="pt-8 border-t border-slate-200 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-slate-400">
            <div>Â© 2026 mocksms. Local sandbox messaging provider. Released under MIT.</div>
            <div className="flex items-center gap-6">
              <span>Local-First</span>
              <span>Â·</span>
              <span>Zero Telemetry</span>
              <span>Â·</span>
              <span>MIT License</span>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
