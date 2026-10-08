/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        ember: {
          50: "#FFF6ED",
          100: "#FFEAD3",
          200: "#FED2A6",
          300: "#FDB26E",
          400: "#FB8C3A",
          500: "#F76E14",
          600: "#E05409",
          700: "#BA3F0A",
          800: "#943311",
          900: "#772D11",
          950: "#401406",
        },
        neutral: {
          0: "#FFFFFF",
          50: "#F7F8FA",
          100: "#EDF0F3",
          200: "#DFE3E8",
          300: "#C6CDD5",
          400: "#9BA5B1",
          500: "#687382",
          600: "#535D6B",
          700: "#3C4552",
          800: "#272E38",
          900: "#181D24",
          950: "#0F1318",
        },
        semantic: {
          success: "#15803D",
          "success-bg": "#DCFCE7",
          info: "#1D4ED8",
          "info-bg": "#DBEAFE",
          warning: "#B45309",
          "warning-bg": "#FEF3C7",
          danger: "#B91C1C",
          "danger-bg": "#FEE2E2",
          inbound: "#0F766E",
          "inbound-bg": "#CCFBF1",
        },
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
      },
      borderRadius: {
        DEFAULT: "8px",
        lg: "12px",
        md: "8px",
        sm: "6px",
        pill: "9999px",
      },
      fontFamily: {
        sans: ["Figtree", "system-ui", "-apple-system", "sans-serif"],
        mono: ['"JetBrains Mono"', "monospace"],
      },
      animation: {
        "arrival-slide": "arrivalSlide 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards",
        "pulse-once": "pulseOnce 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards",
      },
      keyframes: {
        arrivalSlide: {
          "0%": { transform: "translateY(-8px)", opacity: "0.6" },
          "100%": { transform: "translateY(0)", opacity: "1" },
        },
        pulseOnce: {
          "0%": { transform: "scale(0.98)", backgroundColor: "#FFEAD3" },
          "50%": { transform: "scale(1.02)", backgroundColor: "#FED2A6" },
          "100%": { transform: "scale(1)", backgroundColor: "transparent" },
        },
      },
    },
  },
  plugins: [],
};
