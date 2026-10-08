import { type MotionValue, motion, useScroll, useTransform } from "framer-motion";
import {
  CheckCircle2,
  Code2,
  Copy,
  Cpu,
  Flame,
  Radio,
  Send,
  ShieldAlert,
  Smartphone,
  Terminal,
  Zap,
} from "lucide-react";
import { useRef, useState } from "react";

interface Chapter {
  id: number;
  label: string;
  tag: string;
  title: string;
  description: string;
  badge: string;
}

const chapters: Chapter[] = [
  {
    id: 1,
    label: "01. SEND",
    tag: "Native Provider Wire Format",
    title: "You write standard production code.",
    description:
      "Keep your standard Twilio, Termii, or SMTP libraries. Just redirect your client base URL to localhost. No custom mocks, no proprietary SDK wrappers, and no vendor lock-in.",
    badge: "SDK Compatible",
  },
  {
    id: 2,
    label: "02. INTERCEPT",
    tag: "Zero Latency Local Engine",
    title: "We intercept everything hermetically.",
    description:
      "MockSMS catches outbound payloads on your machine before they touch external telecoms. SQLite WAL storage commits in under 1 millisecond. No cloud bills, no test account suspensions.",
    badge: "0ms Interception",
  },
  {
    id: 3,
    label: "03. EXTRACT",
    tag: "Instant OTP & Web Inbox",
    title: "Your team and test runners see it in real-time.",
    description:
      "A live SSE stream renders SMS and emails on the local dashboard. OTP verification codes are parsed with regex intelligence and copied with a single click. End-to-end tests fetch codes via sync wait APIs.",
    badge: "Reactive UI",
  },
  {
    id: 4,
    label: "04. SIMULATE",
    tag: "Chaos & Provider Telemetry",
    title: "Test edge cases you could never simulate live.",
    description:
      "Simulate carrier rate limits, queue backlogs, network timeouts, and magic test phone numbers without burning provider credits. Webhooks fire back to your app with realistic signatures.",
    badge: "Chaos Engine",
  },
];

export function StoryScroll() {
  const containerRef = useRef<HTMLDivElement>(null);
  const [copiedCode, setCopiedCode] = useState(false);

  const { scrollYProgress } = useScroll({
    target: containerRef,
    offset: ["start start", "end end"],
  });

  const scrollToChapter = (chapterIndex: number) => {
    if (!containerRef.current) return;
    const containerTop = containerRef.current.offsetTop;
    const containerHeight = containerRef.current.offsetHeight;
    const targetScroll =
      containerTop + (chapterIndex / 3.5) * (containerHeight - window.innerHeight);
    window.scrollTo({ top: targetScroll, behavior: "smooth" });
  };

  const handleCopyCode = () => {
    setCopiedCode(true);
    setTimeout(() => setCopiedCode(false), 2000);
  };

  return (
    <section
      ref={containerRef}
      className="relative h-[480vh] w-full bg-slate-950 text-white selection:bg-orange-500 selection:text-white"
    >
      {/* Sticky Stage Container */}
      <div className="sticky top-0 h-screen w-full flex flex-col justify-between overflow-hidden px-4 sm:px-6 lg:px-12 py-6 sm:py-8">
        {/* Subtle Background Radial Mesh & Grid */}
        <div className="absolute inset-0 pointer-events-none overflow-hidden opacity-30">
          <div className="absolute top-1/4 left-1/3 w-[600px] h-[600px] bg-orange-600/15 rounded-full blur-[140px]" />
          <div className="absolute bottom-1/4 right-1/4 w-[500px] h-[500px] bg-amber-500/10 rounded-full blur-[120px]" />
          <div
            className="absolute inset-0 opacity-[0.04]"
            style={{
              backgroundImage: "radial-gradient(circle at 1px 1px, #ffffff 1px, transparent 0)",
              backgroundSize: "28px 28px",
            }}
          />
        </div>

        {/* Top Floating Story Stepper Bar */}
        <header className="relative z-20 w-full max-w-5xl mx-auto flex items-center justify-between gap-4 py-2 border-b border-slate-800/80">
          <div className="flex items-center gap-2">
            <span className="flex h-2 w-2 rounded-full bg-orange-500 animate-pulse" />
            <span className="text-xs font-mono font-semibold uppercase tracking-wider text-orange-400">
              Interactive Story
            </span>
          </div>

          {/* Stepper Pills */}
          <nav className="flex items-center gap-1 sm:gap-2">
            {chapters.map((ch, idx) => (
              <ChapterButton
                key={ch.id}
                chapter={ch}
                index={idx}
                scrollYProgress={scrollYProgress}
                onClick={() => scrollToChapter(idx)}
              />
            ))}
          </nav>

          <div className="hidden sm:flex items-center gap-2 text-xs font-mono text-slate-500">
            <span>Scroll to navigate</span>
            <div className="w-4 h-6 border border-slate-700 rounded-full flex items-start justify-center p-1">
              <div className="w-1 h-1.5 bg-orange-400 rounded-full animate-bounce" />
            </div>
          </div>
        </header>

        {/* Main Interactive Stage Split Screen */}
        <main className="relative z-10 w-full max-w-7xl mx-auto flex-1 flex flex-col lg:flex-row items-center justify-center gap-8 lg:gap-14 py-4 min-h-0">
          {/* Left Column: Story Narratives (Dynamic based on scroll) */}
          <div className="w-full lg:w-5/12 flex flex-col justify-center space-y-6 text-left">
            <StoryNarrative scrollYProgress={scrollYProgress} chapters={chapters} />
          </div>

          {/* Right Column: Morphing Visual Studio Sandbox */}
          <div className="w-full lg:w-7/12 h-[380px] sm:h-[460px] md:h-[500px] flex items-center justify-center">
            <div className="relative w-full h-full max-w-2xl rounded-2xl bg-slate-900/90 border border-slate-800 shadow-2xl shadow-black/80 flex flex-col overflow-hidden backdrop-blur-xl">
              {/* Studio Window Chrome Header */}
              <div className="h-10 px-4 bg-slate-950/90 border-b border-slate-800 flex items-center justify-between shrink-0">
                <div className="flex items-center gap-2">
                  <div className="flex items-center gap-1.5">
                    <span className="w-3 h-3 rounded-full bg-red-500/80 inline-block" />
                    <span className="w-3 h-3 rounded-full bg-amber-500/80 inline-block" />
                    <span className="w-3 h-3 rounded-full bg-emerald-500/80 inline-block" />
                  </div>
                  <span className="ml-3 text-xs font-mono text-slate-400 flex items-center gap-1.5">
                    <Cpu className="w-3.5 h-3.5 text-orange-400" />
                    mocksms-sandbox:4010
                  </span>
                </div>

                {/* Status Indicator */}
                <div className="flex items-center gap-2">
                  <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-orange-500/10 text-orange-400 border border-orange-500/20">
                    Live Session
                  </span>
                </div>
              </div>

              {/* Dynamic Screen Content: Layered based on scroll position */}
              <div className="relative flex-1 p-5 overflow-hidden flex items-center justify-center">
                {/* ACT 1: Code Dispatcher */}
                <Act1Screen scrollYProgress={scrollYProgress} />

                {/* ACT 2: Interceptor Engine */}
                <Act2Screen scrollYProgress={scrollYProgress} />

                {/* ACT 3: Live Inbox & Smart OTP */}
                <Act3Screen
                  scrollYProgress={scrollYProgress}
                  copiedCode={copiedCode}
                  onCopyCode={handleCopyCode}
                />

                {/* ACT 4: Chaos & Simulator Engine */}
                <Act4Screen scrollYProgress={scrollYProgress} />
              </div>
            </div>
          </div>
        </main>

        {/* Bottom Interactive Telemetry Bar */}
        <footer className="relative z-20 w-full max-w-5xl mx-auto flex items-center justify-between text-xs font-mono text-slate-400 border-t border-slate-800/80 pt-3">
          <div className="flex items-center gap-4">
            <span className="flex items-center gap-1.5 text-emerald-400">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
              Daemon: Active
            </span>
            <span className="hidden sm:inline text-slate-600">|</span>
            <span className="hidden sm:inline">Port: 4010 (HTTP) &bull; 1025 (SMTP)</span>
          </div>

          <div className="flex items-center gap-3">
            <span className="text-slate-400">Total Billed to Provider:</span>
            <span className="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-bold">
              $0.00
            </span>
          </div>
        </footer>
      </div>
    </section>
  );
}

// ---------------------------------------------------------
// Subcomponents for Narrative & Steppers
// ---------------------------------------------------------

function ChapterButton({
  chapter,
  index,
  scrollYProgress,
  onClick,
}: {
  chapter: Chapter;
  index: number;
  scrollYProgress: MotionValue<number>;
  onClick: () => void;
}) {
  const start = index * 0.25;
  const end = (index + 1) * 0.25;

  // Active opacity calculation
  const isActive = useTransform(
    scrollYProgress,
    [start - 0.05, start, end - 0.05, end],
    [0.4, 1, 1, 0.4]
  );

  return (
    <button
      type="button"
      onClick={onClick}
      className="relative px-2.5 py-1 text-xs font-mono rounded-md transition-colors hover:text-white"
    >
      <motion.span style={{ opacity: isActive }} className="font-semibold text-slate-300">
        {chapter.label}
      </motion.span>
    </button>
  );
}

function StoryNarrative({
  scrollYProgress,
  chapters,
}: {
  scrollYProgress: MotionValue<number>;
  chapters: Chapter[];
}) {
  return (
    <div className="relative h-[220px] sm:h-[260px] w-full">
      {chapters.map((ch, idx) => {
        const start = idx * 0.25;
        const end = (idx + 1) * 0.25;

        // Custom opacity and translation curves per chapter
        const opacity = useTransform(
          scrollYProgress,
          [start - 0.05, start + 0.02, end - 0.04, end],
          [0, 1, 1, 0]
        );
        const y = useTransform(
          scrollYProgress,
          [start - 0.05, start + 0.02, end - 0.04, end],
          [20, 0, 0, -20]
        );

        return (
          <motion.div
            key={ch.id}
            style={{ opacity, y }}
            className="absolute inset-0 flex flex-col justify-center space-y-4"
          >
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-orange-500/10 border border-orange-500/20 text-orange-400 text-xs font-mono font-medium w-fit">
              <Zap className="w-3 h-3" />
              {ch.tag}
            </div>
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-extrabold tracking-tight text-white leading-tight">
              {ch.title}
            </h2>
            <p className="text-base sm:text-lg text-slate-400 leading-relaxed max-w-xl">
              {ch.description}
            </p>
          </motion.div>
        );
      })}
    </div>
  );
}

// ---------------------------------------------------------
// ACT 1: Code Dispatcher Screen
// ---------------------------------------------------------

function Act1Screen({ scrollYProgress }: { scrollYProgress: MotionValue<number> }) {
  const opacity = useTransform(scrollYProgress, [0, 0.02, 0.21, 0.25], [1, 1, 1, 0]);
  const scale = useTransform(scrollYProgress, [0, 0.21, 0.25], [1, 1, 0.95]);

  return (
    <motion.div
      style={{ opacity, scale }}
      className="absolute inset-0 p-4 sm:p-6 flex flex-col justify-between font-mono text-xs sm:text-sm text-slate-300"
    >
      <div className="space-y-2">
        <div className="flex items-center justify-between text-slate-500 text-xs pb-2 border-b border-slate-800">
          <span className="flex items-center gap-1.5">
            <Code2 className="w-3.5 h-3.5 text-orange-400" />
            server/auth_service.ts
          </span>
          <span className="text-slate-600">TypeScript</span>
        </div>

        <pre className="text-slate-300 overflow-x-auto pt-2 leading-relaxed">
          <code>
            <span className="text-purple-400">import</span> twilio{" "}
            <span className="text-purple-400">from</span>{" "}
            <span className="text-emerald-400">"twilio"</span>
            {"\n\n"}
            <span className="text-slate-500">
              {"// Point your existing client to local MockSMS"}
            </span>
            {"\n"}
            <span className="text-blue-400">const</span> client ={" "}
            <span className="text-yellow-400">twilio</span>(
            <span className="text-emerald-400">"AC_DEV"</span>,{" "}
            <span className="text-emerald-400">"AUTH_DEV"</span>, &#123;
            {"\n"}
            {"  "}edge:{" "}
            <span className="text-orange-400 font-bold bg-orange-500/10 px-1 py-0.5 rounded border border-orange-500/30">
              "http://localhost:4010"
            </span>
            {"\n"}&#125;);
            {"\n\n"}
            <span className="text-blue-400">await</span> client.messages.
            <span className="text-yellow-400">create</span>(&#123;
            {"\n"}
            {"  "}to: <span className="text-emerald-400">"+1234567890"</span>,{"\n"}
            {"  "}body: <span className="text-emerald-400">"Your verification code is 589201"</span>
            {"\n"}&#125;);
          </code>
        </pre>
      </div>

      <div className="p-3 rounded-xl bg-orange-500/10 border border-orange-500/20 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Send className="w-4 h-4 text-orange-400 animate-pulse" />
          <span className="text-xs text-orange-200">Dispatched via SDK to local port 4010</span>
        </div>
        <span className="text-[11px] font-bold text-orange-400 bg-orange-500/20 px-2 py-0.5 rounded">
          POST /v1/Messages
        </span>
      </div>
    </motion.div>
  );
}

// ---------------------------------------------------------
// ACT 2: Interceptor Screen
// ---------------------------------------------------------

function Act2Screen({ scrollYProgress }: { scrollYProgress: MotionValue<number> }) {
  const opacity = useTransform(scrollYProgress, [0.23, 0.27, 0.47, 0.5], [0, 1, 1, 0]);
  const scale = useTransform(scrollYProgress, [0.23, 0.27, 0.47, 0.5], [0.95, 1, 1, 0.95]);

  return (
    <motion.div
      style={{ opacity, scale }}
      className="absolute inset-0 p-4 sm:p-6 flex flex-col justify-between font-mono text-xs sm:text-sm text-slate-300"
    >
      <div className="space-y-3">
        <div className="flex items-center justify-between text-slate-500 text-xs pb-2 border-b border-slate-800">
          <span className="flex items-center gap-1.5">
            <Terminal className="w-3.5 h-3.5 text-emerald-400" />
            daemon-stdout (intercept-router)
          </span>
          <span className="text-emerald-400 flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping" />
            Interception Active
          </span>
        </div>

        {/* Real-time Simulated Terminal Stream */}
        <div className="space-y-2 text-xs leading-relaxed">
          <div className="text-slate-400">
            <span className="text-purple-400">12:00:04.102</span> [HTTP] POST
            /2010-04-01/Accounts/AC_DEV/Messages.json
          </div>
          <div className="text-emerald-400 flex items-center gap-2">
            <span>&rarr;</span>
            <span>Intercepted inbound wire call. Blocked external network access.</span>
          </div>
          <div className="text-slate-300 bg-slate-950 p-2.5 rounded-lg border border-slate-800 space-y-1">
            <div className="text-slate-400">
              MsgID: <span className="text-orange-400">msg_01J9X8B7T0M90W1</span>
            </div>
            <div className="text-slate-400">
              Recipient: <span className="text-cyan-400">+1234567890</span> (E.164 verified)
            </div>
            <div className="text-slate-400">
              Encoding: <span className="text-emerald-400">GSM-7</span> (1 segment, 34 chars)
            </div>
          </div>
          <div className="text-slate-400">
            <span className="text-purple-400">12:00:04.103</span> [SQLite] Transacted into memory
            WAL in <span className="text-orange-400">0.42ms</span>
          </div>
        </div>
      </div>

      <div className="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 text-emerald-400" />
          <span className="text-xs text-emerald-200">
            Carrier bypassed: 100% offline & zero cost
          </span>
        </div>
        <span className="text-[11px] font-bold text-emerald-400 bg-emerald-500/20 px-2 py-0.5 rounded">
          HTTP 201 Created
        </span>
      </div>
    </motion.div>
  );
}

// ---------------------------------------------------------
// ACT 3: Smart Live Inbox & OTP Screen
// ---------------------------------------------------------

function Act3Screen({
  scrollYProgress,
  copiedCode,
  onCopyCode,
}: {
  scrollYProgress: MotionValue<number>;
  copiedCode: boolean;
  onCopyCode: () => void;
}) {
  const opacity = useTransform(scrollYProgress, [0.48, 0.52, 0.72, 0.75], [0, 1, 1, 0]);
  const scale = useTransform(scrollYProgress, [0.48, 0.52, 0.72, 0.75], [0.95, 1, 1, 0.95]);

  return (
    <motion.div
      style={{ opacity, scale }}
      className="absolute inset-0 p-4 sm:p-6 flex flex-col justify-between"
    >
      <div className="space-y-4">
        <div className="flex items-center justify-between text-slate-500 text-xs pb-2 border-b border-slate-800">
          <span className="flex items-center gap-1.5 font-mono">
            <Radio className="w-3.5 h-3.5 text-orange-400 animate-pulse" />
            Live SSE Event Stream: message.created
          </span>
          <span className="text-orange-400 font-mono text-xs">0ms Latency</span>
        </div>

        {/* Message Card Simulation */}
        <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-3">
          <div className="flex items-center justify-between text-xs">
            <span className="font-semibold text-white flex items-center gap-1.5">
              <Smartphone className="w-3.5 h-3.5 text-orange-400" />
              +1 234 567 890
            </span>
            <span className="text-slate-500 font-mono">Just now</span>
          </div>

          <p className="text-sm text-slate-300">
            Your verification code is{" "}
            <span className="font-bold text-white bg-slate-800 px-1.5 py-0.5 rounded">589201</span>.
            Valid for 10 minutes.
          </p>

          {/* 1-Click Code Tile */}
          <div className="pt-2 flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="text-xs font-mono text-slate-400">Extracted OTP:</span>
              <button
                type="button"
                onClick={onCopyCode}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-orange-500 text-white font-mono font-bold text-xs shadow-md shadow-orange-500/20 hover:bg-orange-600 transition-all active:scale-95"
              >
                {copiedCode ? (
                  <>
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    Copied!
                  </>
                ) : (
                  <>
                    <Copy className="w-3.5 h-3.5" />
                    589201
                  </>
                )}
              </button>
            </div>

            <span className="text-[11px] font-mono text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20">
              Status: Delivered
            </span>
          </div>
        </div>
      </div>

      {/* Sync Test API Wait Box */}
      <div className="p-3 rounded-xl bg-slate-950 border border-slate-800 font-mono text-xs flex items-center justify-between text-slate-400">
        <span className="flex items-center gap-2">
          <span className="text-cyan-400">GET</span> /api/v1/otp/latest?to=+1234567890
        </span>
        <span className="text-emerald-400">&rarr; 589201</span>
      </div>
    </motion.div>
  );
}

// ---------------------------------------------------------
// ACT 4: Chaos & Simulator Engine Screen
// ---------------------------------------------------------

function Act4Screen({ scrollYProgress }: { scrollYProgress: MotionValue<number> }) {
  const opacity = useTransform(scrollYProgress, [0.73, 0.77, 0.98, 1], [0, 1, 1, 1]);
  const scale = useTransform(scrollYProgress, [0.73, 0.77, 0.98, 1], [0.95, 1, 1, 1]);

  return (
    <motion.div
      style={{ opacity, scale }}
      className="absolute inset-0 p-4 sm:p-6 flex flex-col justify-between font-mono text-xs sm:text-sm text-slate-300"
    >
      <div className="space-y-4">
        <div className="flex items-center justify-between text-slate-500 text-xs pb-2 border-b border-slate-800">
          <span className="flex items-center gap-1.5 text-amber-400 font-bold">
            <ShieldAlert className="w-3.5 h-3.5" />
            Failure & Latency Simulation Engine
          </span>
          <span className="text-xs bg-amber-500/10 text-amber-400 px-2 py-0.5 rounded border border-amber-500/20">
            Chaos Rules: ON
          </span>
        </div>

        {/* Chaos Rule Cards */}
        <div className="space-y-2">
          <div className="p-3 rounded-lg bg-slate-950 border border-slate-800 flex items-center justify-between">
            <div>
              <div className="text-xs font-bold text-white">Magic Pattern 999901</div>
              <div className="text-[11px] text-slate-500">Number ends in ...999901</div>
            </div>
            <span className="text-xs font-bold text-red-400 bg-red-500/10 px-2 py-0.5 rounded border border-red-500/20">
              Error 21211 (Invalid)
            </span>
          </div>

          <div className="p-3 rounded-lg bg-slate-950 border border-slate-800 flex items-center justify-between">
            <div>
              <div className="text-xs font-bold text-white">Carrier Outage Simulation</div>
              <div className="text-[11px] text-slate-500">Synthetic 2,000ms delay + 500 error</div>
            </div>
            <span className="text-xs font-bold text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/20">
              Error 30008 (Timeout)
            </span>
          </div>

          <div className="p-3 rounded-lg bg-slate-950 border border-slate-800 flex items-center justify-between">
            <div>
              <div className="text-xs font-bold text-white">Webhook Delivery Replay</div>
              <div className="text-[11px] text-slate-500">
                Dispatches status callback with signature
              </div>
            </div>
            <span className="text-xs font-bold text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20">
              POST /webhook (200 OK)
            </span>
          </div>
        </div>
      </div>

      <div className="p-3 rounded-xl bg-orange-500/10 border border-orange-500/20 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Flame className="w-4 h-4 text-orange-400" />
          <span className="text-xs text-orange-200">
            Bulletproof your notification retries before release
          </span>
        </div>
        <span className="text-[11px] font-bold text-orange-400 bg-orange-500/20 px-2 py-0.5 rounded">
          Deterministic
        </span>
      </div>
    </motion.div>
  );
}
