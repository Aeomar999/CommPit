import { type MotionValue, motion, useScroll, useTransform } from "framer-motion";
import { Code, ShieldCheck, Terminal, Zap } from "lucide-react";
import { useRef } from "react";

export function StoryScroll() {
  const targetRef = useRef<HTMLDivElement>(null);

  // Track scroll progress relative to this container
  // It's 400vh tall, so it takes 4 screen heights to scroll past
  const { scrollYProgress } = useScroll({
    target: targetRef,
    offset: ["start start", "end end"],
  });

  // Background color interpolation for dramatic cinematic effect
  const bg = useTransform(
    scrollYProgress,
    [0, 0.2, 0.4, 0.6, 0.8, 1],
    [
      "#ffffff", // Start white
      "#f8fafc", // Slate 50
      "#0f172a", // Deep slate 900 for dark mode scene
      "#020617", // Slate 950
      "#fff7ed", // Orange 50 for vibrant finale
      "#ffffff", // Back to white to blend with next section
    ]
  );

  return (
    <motion.section
      ref={targetRef}
      style={{ backgroundColor: bg }}
      className="relative h-[400vh] w-full"
    >
      {/* Sticky container that holds the scene */}
      <div className="sticky top-0 h-screen w-full overflow-hidden flex items-center justify-center">
        {/* Background Ambient Glows */}
        <AmbientGlows progress={scrollYProgress} />

        {/* The Scenes */}
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none z-10">
          <Scene1 progress={scrollYProgress} />
          <Scene2 progress={scrollYProgress} />
          <Scene3 progress={scrollYProgress} />
          <Scene4 progress={scrollYProgress} />
        </div>
      </div>
    </motion.section>
  );
}

function AmbientGlows({ progress }: { progress: MotionValue<number> }) {
  // A glowing orb that moves and changes color
  const x = useTransform(progress, [0, 1], ["-50%", "50%"]);
  const y = useTransform(progress, [0, 1], ["-20%", "20%"]);
  const opacity = useTransform(progress, [0, 0.5, 1], [0, 0.5, 0]);

  return (
    <motion.div
      style={{ x, y, opacity }}
      className="absolute w-[800px] h-[800px] rounded-full blur-[120px] bg-orange-500/20 mix-blend-screen pointer-events-none"
    />
  );
}

function Scene1({ progress }: { progress: MotionValue<number> }) {
  // Active from 0.0 to 0.25
  const opacity = useTransform(progress, [0, 0.05, 0.15, 0.25], [0, 1, 1, 0]);
  const y = useTransform(progress, [0, 0.05, 0.15, 0.25], [100, 0, 0, -100]);
  const scale = useTransform(progress, [0, 0.05, 0.15, 0.25], [0.9, 1, 1, 1.1]);

  return (
    <motion.div
      style={{ opacity, y, scale }}
      className="absolute flex flex-col items-center text-center max-w-3xl px-6"
    >
      <div className="w-24 h-24 bg-white border border-slate-200 rounded-3xl flex items-center justify-center mb-10 shadow-xl shadow-slate-200/50">
        <Code className="w-12 h-12 text-slate-800" strokeWidth={1.5} />
      </div>
      <h2 className="text-5xl md:text-7xl font-bold text-slate-900 mb-8 tracking-tighter">
        You write the code.
      </h2>
      <p className="text-xl md:text-2xl text-slate-600 font-medium leading-relaxed max-w-2xl mx-auto">
        Keep your existing Twilio or Termii SDKs. Just change the base URL to{" "}
        <code className="px-2 py-1 bg-slate-100 rounded-md text-orange-600 font-mono text-lg mx-1">
          localhost:4010
        </code>
        . No code rewrites.
      </p>
    </motion.div>
  );
}

function Scene2({ progress }: { progress: MotionValue<number> }) {
  // Active from 0.25 to 0.5
  const opacity = useTransform(progress, [0.2, 0.3, 0.4, 0.5], [0, 1, 1, 0]);
  const y = useTransform(progress, [0.2, 0.3, 0.4, 0.5], [100, 0, 0, -100]);
  const scale = useTransform(progress, [0.2, 0.3, 0.4, 0.5], [0.9, 1, 1, 1.1]);

  return (
    <motion.div
      style={{ opacity, y, scale }}
      className="absolute flex flex-col items-center text-center max-w-3xl px-6"
    >
      <div className="w-24 h-24 bg-slate-800 border border-slate-700 rounded-3xl flex items-center justify-center mb-10 shadow-2xl shadow-slate-900/50">
        <Terminal className="w-12 h-12 text-emerald-400" strokeWidth={1.5} />
      </div>
      <h2 className="text-5xl md:text-7xl font-bold text-white mb-8 tracking-tighter">
        We catch it instantly.
      </h2>
      <p className="text-xl md:text-2xl text-slate-400 font-medium leading-relaxed max-w-2xl mx-auto">
        MockSMS intercepts your payloads directly on your machine. Zero network latency, zero API
        costs, and absolute privacy.
      </p>
    </motion.div>
  );
}

function Scene3({ progress }: { progress: MotionValue<number> }) {
  // Active from 0.5 to 0.75
  const opacity = useTransform(progress, [0.45, 0.55, 0.65, 0.75], [0, 1, 1, 0]);
  const y = useTransform(progress, [0.45, 0.55, 0.65, 0.75], [100, 0, 0, -100]);
  const scale = useTransform(progress, [0.45, 0.55, 0.65, 0.75], [0.9, 1, 1, 1.1]);

  return (
    <motion.div
      style={{ opacity, y, scale }}
      className="absolute flex flex-col items-center text-center max-w-3xl px-6"
    >
      <div className="w-24 h-24 bg-gradient-to-br from-orange-500 to-amber-500 rounded-3xl flex items-center justify-center mb-10 shadow-2xl shadow-orange-500/30">
        <Zap className="w-12 h-12 text-white fill-white/20" strokeWidth={1.5} />
      </div>
      <h2 className="text-5xl md:text-7xl font-bold text-white mb-8 tracking-tighter">
        Real-time feedback.
      </h2>
      <p className="text-xl md:text-2xl text-slate-400 font-medium leading-relaxed max-w-2xl mx-auto">
        Watch messages appear in the live inbox in milliseconds. Verify OTP extraction, inspect
        headers, and ensure perfect delivery before production.
      </p>
    </motion.div>
  );
}

function Scene4({ progress }: { progress: MotionValue<number> }) {
  // Active from 0.75 to 1.0
  const opacity = useTransform(progress, [0.7, 0.8, 0.95, 1], [0, 1, 1, 0]);
  const y = useTransform(progress, [0.7, 0.8, 0.95, 1], [100, 0, 0, -100]);
  const scale = useTransform(progress, [0.7, 0.8, 0.95, 1], [0.9, 1, 1, 1.1]);

  return (
    <motion.div
      style={{ opacity, y, scale }}
      className="absolute flex flex-col items-center text-center max-w-3xl px-6"
    >
      <div className="w-24 h-24 bg-white border border-slate-200 rounded-3xl flex items-center justify-center mb-10 shadow-xl shadow-slate-200/50">
        <ShieldCheck className="w-12 h-12 text-orange-600" strokeWidth={1.5} />
      </div>
      <h2 className="text-5xl md:text-7xl font-bold text-slate-900 mb-8 tracking-tighter">
        Test the impossible.
      </h2>
      <p className="text-xl md:text-2xl text-slate-600 font-medium leading-relaxed max-w-2xl mx-auto">
        Simulate carrier outages, rate limits, and async webhook failures locally. Build resilient
        systems without paying for real failed messages.
      </p>
    </motion.div>
  );
}
