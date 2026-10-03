# mocksms — Design

The visual and interaction design for every mocksms surface: the web inbox (built first, in M1), the documentation site, and the landing page. It turns the product truth in [PRODUCT.md](../PRODUCT.md) into a system the build follows.

**Status:** direction approved 2026-10-03. No UI is built yet. Impeccable's root `DESIGN.md` (with its `.impeccable/design.json` sidecar) is generated from the built inbox at the end of M1 by its documenter; until then, this file is the design authority. If the two ever disagree after M1, the generated `DESIGN.md` describes what shipped and this file is updated to match.

---

## 1. How this was decided

| Skill | What it contributed |
|---|---|
| **impeccable** | The process: product interview → `PRODUCT.md`, a direction round with challengers, surface modes (Operate / Read / Persuade), the craft floor (§13), and the finish workflow (§15). |
| **design-taste-frontend** ("taste") | The design read and dials (§2), anti-default checks, the em-dash ban in UI copy, icon-library choice, and the landing-page pre-flight (§12.3). Its own scope excludes dense app UI, so it governs the landing page fully and the inbox only through its general rules. |
| **ui-ux-pro-max** | Category baseline (what a neutral inbox defaults to), the accessibility, interaction and animation checklists (§11, §14), and the pre-delivery checklist. |
| **design:design-system** | The token taxonomy (§5 to §9) and the component documentation format (§10): variants, states, accessibility, do and don't. |

**The decision trail.** Two impeccable direction rounds were dealt (a transit line map and a live league table led them, with telegram, scratch-card, boarding-pass and ticket-wallet alternates). The project owner then pinned a direction directly with reference screenshots of a warm, orange-accented bulk-SMS dashboard and landing page. A pinned direction beats the roll, so that reference is the visual world; useful disciplines from the rounds are kept as system rules (§4.3).

---

## 2. Design read

> **Reading this as:** a local developer tool (inbox, test console and inspector) for developers and QA engineers, in a warm, friendly, light SaaS language with one ember-orange accent, built on Tailwind v4 + customised shadcn/ui, Figtree and JetBrains Mono.

**Physical scene.** A developer at a laptop in a bright office, café or home by day, mocksms open in a browser window beside their editor and terminal, glancing at it dozens of times an hour; at night, the same developer with a dark editor. So: **light is designed first** (it matches the pinned reference and daytime use), **dark follows the system setting** and is finished to the same standard, and a manual toggle is available.

**Dials** (taste):

| Surface | Variance | Motion | Density | Why |
|---|---|---|---|---|
| Inbox (Operate) | 3 | 3 | 5 | Predictable app layout; motion only for arrival and state; comfortable density as chosen |
| Docs (Read) | 3 | 2 | 4 | A document first; calm column |
| Landing (Persuade) | 6 | 5 | 4 | Room for one bold composition and a live demo |

---

## 3. Direction

### 3.1 The pinned reference

The owner's reference is a production bulk-SMS dashboard and its marketing page: light grey ground, white rounded cards, one bright orange accent, a left sidebar with soft icons, stat cards with a big number, segmented tabs, generous spacing, friendly empty states, a "get started" checklist, and a landing page with an orange statement band and a dark API code block.

### 3.2 What we take, what we leave

| Take | Leave |
|---|---|
| Light, airy ground with white rounded surfaces | The reference product's name, logo, wordmark, copy and exact brand orange. mocksms has its own identity (§3.4) |
| One warm orange doing real jobs: primary action, active navigation, icon tiles, highlights | White text on bright orange (fails WCAG AA; see §5.4) |
| Sidebar navigation with soft duotone icons and a clear active state | The floating support-chat button (there is no support chat in a local tool) |
| Segmented tabs, page-level actions top right | Thick coloured left borders on callouts (craft floor ban) |
| Stat tiles: tinted icon tile + label + large figure | Stat tiles as the default page structure: here they carry codes and counts only where those are the content (§4.2) |
| Empty states with an illustration-free, clear next action | Small labels above section headings on the landing page (craft floor ban) |
| A progress checklist for first-run onboarding | |
| Landing: orange statement band, dark API code block, orange closing band | Invented proof (customer logos, testimonials, delivery percentages): mocksms has none yet |

### 3.3 Never mistaken for a live provider

Because the reference is a real messaging provider's dashboard, mocksms must never be mistaken for one. Three rules:

1. A persistent **Sandbox pill** in the top bar: "Sandbox: nothing is delivered" (§10.12).
2. Delivery language stays honest: "Delivered (simulated)" in tooltips and detail views.
3. Distinct identity: own ember orange (redder than the reference's amber-orange), own wordmark, own signature move (§4.2).

### 3.4 Identity

- **Name:** mocksms, lowercase.
- **Interim wordmark:** "mocksms" set in Figtree 700, `mock` in neutral-900 and `sms` in ember-700 (light) / ember-400 (dark).
- **Mark concept (to be designed with the logo-design skill before the landing page ships):** a speech bubble drawn in a dashed stroke, a message that is never really sent. The dashed stroke is the only place "dashed" appears in the system besides the Sandbox pill.

---

## 4. The system in one page

### 4.1 Principles

1. **The code is the point.** OTPs, links and IDs are what people came for: lifted out, set large, one click to copy.
2. **Warm, never cute.** Friendly copy and soft shapes; precise data. No mascots, no emoji.
3. **Show the raw truth.** Raw requests, MIME and webhook payloads are one click away and styled as first-class content.
4. **Status is never colour alone.** Every status carries an icon and a word.
5. **Live, calmly.** Things arrive in real time, announce themselves once, and stay put.

### 4.2 Signature move: the code tile

Any OTP, verification code or verification link a message carries is lifted out of the text into a **code tile**, built in the reference's stat-tile language: a soft ember tile, the code set large in JetBrains Mono with tabular figures, a copy button, its source ("Verify" or "found in message"), and a countdown when it expires.

- Appears at three scales: compact in the inbox row, large in the message detail header, and as a grid in the OTPs view.
- On arrival the tile pulses once in ember and keeps a small "new" marker until the message is opened.
- Links get the same tile with the URL's host and path, an "Open" and a "Copy" button.
- The tile is the one place the stat-tile pattern is used for content; counts elsewhere use it sparingly (project overview, estimate).

### 4.3 Disciplines kept from the direction rounds

| Rule | Applied as |
|---|---|
| **New stays lit until seen** | A newly arrived or status-changed row keeps an ember dot and a faint ember-50 ground until opened |
| **Retired stays visible** | Unsubscribed numbers, expired codes and pruned items grey out in place with a reason, never vanish silently |
| **Foreign media in a plate** | Email HTML always renders inside a bordered plate (§10.8); the sender's styles never touch the app |
| **Tabular discipline** | Every number (times, counts, segments, durations, codes) uses tabular figures in fixed columns, so live updates never jitter |
| **One tonal ramp** | Every neutral comes from the single 11-step ramp in §5.2; no ad-hoc greys |
| **Density courage for batches** | A 10,000-recipient batch renders as a field of status cells, not a progress bar (§10.11) |
| **Focus with context** | The selected conversation brightens; the rest of the list stays readable, never collapsed |

---

## 5. Colour

### 5.1 Strategy

**Restrained, committed accent:** tinted cool neutrals plus one ember orange that does real jobs (primary action, active navigation, selection, new-item highlight, code tiles, focus). Status colours are semantic and always paired with icon and label. One accent across all surfaces; no second brand colour.

### 5.2 Ramps

**Ember** (accent):

| Step | Hex | Main use |
|---|---|---|
| 50 | `#FFF6ED` | New-item ground, icon tile ground |
| 100 | `#FFEAD3` | Selected row ground, hover on tiles |
| 200 | `#FED2A6` | Text selection (light) |
| 300 | `#FDB26E` | Decorative strokes |
| 400 | `#FB8C3A` | Primary hover; accent text on dark |
| 500 | `#F76E14` | **Primary fill**, code-tile accent, arrival pulse |
| 600 | `#E05409` | Primary pressed; focus ring (light); large text on white |
| 700 | `#BA3F0A` | **Accent text on light** (links, active nav) |
| 800 | `#943311` | Text selection (dark) |
| 900 | `#772D11` | |
| 950 | `#401406` | Ink on ember fills |

**Neutral** (the one tonal ramp, cool):

| Step | Hex | Light theme use | Dark theme use |
|---|---|---|---|
| 0 | `#FFFFFF` | Surfaces (cards, panes) | |
| 50 | `#F7F8FA` | App ground | Primary text |
| 100 | `#EDF0F3` | Card borders, table header ground | |
| 200 | `#DFE3E8` | Dividers, input borders | |
| 300 | `#C6CDD5` | Disabled borders, scrollbar thumb | Secondary text |
| 400 | `#9BA5B1` | Icons (inactive), placeholder at ≥16px only | Muted text, inactive icons |
| 500 | `#687382` | Muted text (4.8:1 on white) | |
| 600 | `#535D6B` | Secondary text | |
| 700 | `#3C4552` | | Borders on raised surfaces, scrollbar thumb |
| 800 | `#272E38` | | Raised surfaces, borders |
| 900 | `#181D24` | Primary text | Surfaces |
| 950 | `#0F1318` | | App ground |

### 5.3 Semantic and status colours

| Role | Light text | Light ground | Dark text | Dark ground |
|---|---|---|---|---|
| Success | `#15803D` | `#DCFCE7` | `#4ADE80` | `rgb(34 197 94 / 0.14)` |
| Info | `#1D4ED8` | `#DBEAFE` | `#60A5FA` | `rgb(59 130 246 / 0.16)` |
| Warning | `#B45309` | `#FEF3C7` | `#FBBF24` | `rgb(245 158 11 / 0.14)` |
| Danger | `#B91C1C` | `#FEE2E2` | `#F87171` | `rgb(239 68 68 / 0.14)` |
| Inbound | `#0F766E` | `#CCFBF1` | `#2DD4BF` | `rgb(20 184 166 / 0.14)` |

**Message status mapping** (icons from Phosphor):

| Status | Role | Icon | Label |
|---|---|---|---|
| `queued` | Neutral | `Clock` | Queued |
| `sent` | Info | `PaperPlaneTilt` | Sent |
| `delivered` | Success | `Checks` | Delivered |
| `undelivered` | Warning | `WarningCircle` | Undelivered |
| `failed` | Danger | `XCircle` | Failed |
| `received` | Inbound | `ArrowBendDownLeft` | Received |

**Verification status mapping:** `pending` Warning `Hourglass` "Waiting for code"; `approved` Success `SealCheck` "Approved"; `expired` Neutral, greyed `ClockCountdown` "Expired"; `canceled` Neutral, struck `Prohibit` "Canceled"; `max_attempts` Danger `LockKey` "Too many attempts".

### 5.4 Contrast rules (checked values)

| Pair | Ratio | Verdict |
|---|---|---|
| Neutral-900 on white | ≈17:1 | Body text |
| Neutral-600 on white | 6.7:1 | Secondary text |
| Neutral-500 on white | 4.8:1 | Muted text (smallest allowed for body) |
| Ember-700 on white | 5.5:1 | Accent text, links, active nav |
| Ember-950 ink on ember-500 fill | ≈6.3:1 | **Primary button label** |
| White on ember-500 | 2.9:1 | **Never** for text |
| White on ember-600 | 3.9:1 | Large text only (≥24px, or ≥19px bold): landing band |
| Ember-600 ring on white | 3.9:1 | Focus indicator (≥3:1 non-text) |
| Status text on its ground (all five) | 4.5 to 5.5:1 | Chips |
| Ember-400 on neutral-900 (dark) | 7.2:1 | Accent text in dark |

Every new colour pairing gets its ratio checked before it ships (§14).

---

## 6. Typography

### 6.1 Faces

| Face | Use | Why |
|---|---|---|
| **Figtree** (variable, 300 to 900) | All UI text, headings, docs body, landing headlines | Friendly geometric sans in the spirit of the reference, open counters, excellent legibility; not one of the overused defaults |
| **JetBrains Mono** (variable) | Codes, SIDs, phone numbers in code tiles, payloads, headers, code blocks | Unambiguous `0/O`, `1/l/I`; slashed zero (`font-feature-settings: "zero"`); ligatures off |

Both are self-hosted through Fontsource (`@fontsource-variable/figtree`, `@fontsource-variable/jetbrains-mono`) with `font-display: swap`. Never a hosted font stylesheet. Monospace is for code, data and measurement only, never as a "technical" costume for headings or labels.

Figtree numbers use `font-variant-numeric: tabular-nums` in tables, lists and counters. Confirm the shipped Figtree build exposes `tnum` during M1-17; if it does not, numeric columns switch to JetBrains Mono.

### 6.2 Scale

| Token | Size / line height | Weight | Use |
|---|---|---|---|
| `display-xl` | 56 / 60 | 700 | Landing headline (desktop) |
| `display` | 36 / 42 | 700 | Landing section headings, docs page titles |
| `title` | 24 / 32 | 600 | Page titles in the app |
| `heading` | 18 / 26 | 600 | Card and pane headings |
| `subheading` | 15 / 22 | 600 | Group labels, table headers |
| `body` | 14 / 22 | 400 | App body (desktop) |
| `body-lg` | 16 / 26 | 400 | Docs body, landing body, all mobile body and inputs |
| `small` | 13 / 20 | 400 / 500 | Metadata, helper text |
| `caption` | 12 / 16 | 500 | Timestamps, badges (never body copy) |
| `code-hero` | 36 / 40 mono | 600 | Code tile, large |
| `code-lg` | 20 / 28 mono | 600 | Code tile in lists and grids |
| `code` | 13 / 20 mono | 400 | Payloads, headers, inline code |

Rules: headings tracked −0.01em (display −0.02em; never below −0.04em); body measure 65 to 75 characters on reading surfaces; more space above a heading than below it; weights step 400 → 500 → 600 → 700, never 800+ in the app.

---

## 7. Space, shape, elevation

**Spacing:** 4px base. Tokens 1 (4), 2 (8), 3 (12), 4 (16), 5 (20), 6 (24), 8 (32), 10 (40), 12 (48), 16 (64). Every element snaps to the 4px grid.

**Comfortable density** (as chosen): inbox rows 72px; table rows 52px; card padding 20 to 24px; page padding 24px (desktop) and 16px (mobile); gaps 16 between related items, 24 between groups, 32 between sections.

**Shape lock:** containers (cards, panes, dialogs, plates) 12px; controls (buttons, inputs, segmented tabs, menus) 8px with 6px inner segments; chips, badges and the Sandbox pill full pill; code tiles 12px. No other radii.

**Elevation:**

| Level | Light | Dark | Use |
|---|---|---|---|
| 0 | Flat, no border | Flat | App ground |
| 1 | White, 1px neutral-100 border, no shadow | Neutral-900, 1px neutral-800 border | Cards, panes, tables |
| 2 | `0 2px 6px rgb(24 29 36 / 0.06), 0 8px 24px -6px rgb(24 29 36 / 0.12)` | Neutral-800 surface + `0 8px 24px -6px rgb(0 0 0 / 0.5)` | Menus, popovers, toasts |
| 3 | Level 2 shadow doubled, scrim `rgb(15 19 24 / 0.45)` | Same, scrim `rgb(0 0 0 / 0.6)` | Dialogs, sheets |

Shadows always carry an offset and blur; no coloured halos, no glass, no gradients on surfaces.

**Z-index scale:** base 0, sticky 10, sidebar 20, dropdown 30, overlay 40, dialog 50, toast 60. Nothing else.

---

## 8. Iconography and imagery

- **Library:** Phosphor (`@phosphor-icons/react`), replacing lucide in every shadcn component we copy in.
- **Weights:** regular (1.5px look) for UI; duotone for sidebar navigation and icon tiles, echoing the reference's soft filled icons; fill for the active navigation item, in ember.
- **Sizes:** 16 (inline, chips), 20 (buttons, inputs, rows), 24 (navigation, tiles). Icon tiles are 40×40 (rows) and 48×48 (stat and code tiles).
- **Never:** emoji or Unicode glyphs as icons, hand-drawn icon paths, mixed libraries.
- **Imagery:** the app has none. Empty states use typography and a single icon tile, not illustrations. Provider names appear as text chips until the legal check on provider marks (task X-02) clears logos.

---

## 9. Motion

| Token | Value | Use |
|---|---|---|
| `fast` | 120ms | Hover, press, colour changes |
| `base` | 180ms | Menus, tabs, status-track fill, toasts entering |
| `slow` | 260ms | Sheets, dialogs, pane transitions |
| `arrival` | 600ms, once | The authored moment (below) |
| `ease-out` | `cubic-bezier(0.16, 1, 0.3, 1)` | Everything entering or changing |
| `ease-in` | `cubic-bezier(0.7, 0, 0.84, 0)` | Exits (60 to 70% of enter duration) |

**The one authored moment: arrival.** A new message slides 8px down into the top of the list from an already-visible state, its ground blooms ember-50, its code tile pulses once in ember-500, and the "new" dot stays until it is opened. Status changes advance the delivery track's fill (`base`). Nothing else animates on its own: no looping effects, no entrance animation on every section.

**Press:** buttons scale to 0.98 on `:active`. **Copy:** the copy icon morphs to a check with "Copied" for 1.2s.

**Reduced motion:** arrival becomes a colour change only; slides and pulses are removed; durations drop to `fast` or zero. Animations are interruptible and never block input. Only `transform`, `opacity`, `background-color`, `box-shadow` and `clip-path` animate.

---

## 10. Components

Built on shadcn/ui, always customised to these tokens (never shipped in default state). Each entry follows the design-system format.

### 10.1 Button

| Variant | Look | Use |
|---|---|---|
| Primary | Ember-500 fill, ember-950 label, 600 weight | One per view: the main action ("Send test message", "Copy SMTP settings") |
| Secondary | White fill, 1px neutral-200 border, neutral-900 label | Supporting actions |
| Ghost | No fill, neutral-600 label, neutral-100 hover | Toolbar and row actions |
| Destructive | Danger-text label on white, danger ground on hover; confirm dialog | "Clear inbox", "Reset project" |

**Sizes:** sm 32px, md 40px (default), lg 48px (landing). **States:** default, hover (ember-400 / neutral-50), active (ember-600, scale 0.98), focus-visible (2px ember-600 ring, 2px offset), disabled (40% opacity, `cursor: not-allowed`, `aria-disabled`), loading (spinner replaces icon, label stays, width fixed). **Rules:** labels name the action, one line at desktop, three words at most for primary; never two buttons with the same intent on one view.

### 10.2 Segmented tabs

The reference's page-level switcher: a neutral-100 track (8px radius) with segments; the active segment is white with neutral-900 text and a level-1 border (light) or neutral-800 (dark). Ember is not used here, so the primary button stays the only ember action on the page. Keyboard: arrow keys move, `Home`/`End`, `role="tablist"`.

### 10.3 Sidebar navigation

248px wide, collapsible to a 72px icon rail (state remembered per browser). Wordmark at the top; groups **Inbox** (All, SMS, Email), **OTPs**, **Batches**, **Inspector** (Requests, Webhooks), **Estimate**, **Settings**. Bottom: project switcher, connection details (`HTTP 127.0.0.1:4010`, `SMTP :1025`) and theme toggle.

Items: 40px tall, duotone icon + label. **Active:** ember-700 label, fill-weight ember icon, white ground (light) / neutral-800 (dark), and a 3px ember-500 indicator bar inside the item's left edge (an indicator inside the nav item, not a border on a card). Unread counts as caption-size pills. `aria-current="page"` on the active item.

### 10.4 Top bar

64px. Left: page title (`title`). Right: global search (opens with `Ctrl/⌘ K`), today's counters as a compact pill ("128 messages · 3 failed", tabular), the **Sandbox pill** (§10.12), and a docs link. Sticky, level-1 bottom border.

### 10.5 Message row (inbox list)

72px, three lines of information in two rows:

```
[tile] +233 24 123 4567            Twilio   14:02   ● Delivered
       Your Acme code is 482913     [ 482 913  copy ]
```

- Channel tile: SMS (ember-50 tile, `ChatText`), email (neutral-100 tile, `EnvelopeSimple`).
- Recipient in `subheading`; provider as a neutral text chip; time relative ("2m") with the absolute time in a tooltip; status chip.
- Second line: preview in neutral-600, then the compact code tile when a code was found.
- **New:** ember dot before the recipient + ember-50 ground until opened. **Selected:** ember-100 ground, others unchanged. **Retired** (unsubscribed number, expired code): neutral-400 text with a reason chip.
- Keyboard: `j`/`k` or arrows move, `Enter` opens, `c` copies the row's code (single-key shortcuts can be turned off in Settings, per WCAG 2.1.4).

### 10.6 Code tile (signature)

| Size | Contents | Where |
|---|---|---|
| Compact | Code in `code-lg` (digits grouped in threes), copy icon button | Inbox rows |
| Large | Ember-50 tile, `code-hero` code, source label ("Verify" / "Found in message"), countdown ("Expires in 9:41"), Copy button | Message detail header |
| Grid | Large tile + recipient + provider + status | OTPs view |

States: new (one ember-500 pulse, "new" dot), copied (check + "Copied"), expired (greyed, struck through, "Expired 2m ago"), approved (success chip). Accessibility: the code is real text (selectable), the button is labelled "Copy code 482913", and grouping spaces are visual only (`aria-label` carries the ungrouped digits).

### 10.7 Delivery track

A four-step horizontal track inside message detail: Queued → Sent → Delivered, with the final step swapping to Undelivered or Failed (icon + label + provider error code, e.g. "30005 Unknown handset"). Completed steps fill in the status colour; timestamps under each step in `caption`, tabular. Webhook attempts hang beneath the step that triggered them (attempt number, response code, "Replay").

### 10.8 Email plate

Email HTML renders in a sandboxed iframe (`sandbox` without `allow-scripts`) set inside a **plate**: a 12px container with a 1px neutral-200 border and a neutral-50 surround, with a slim header: "Rendered in a sandbox. Scripts are off and remote images are blocked." plus a "Load images" button. Tabs above the plate: HTML, Text, Source, Headers, Attachments (segmented tabs). Found links and codes appear as code tiles above the plate.

### 10.9 Status chip

Pill, `caption` 500 weight, 6px × 10px padding, semantic ground + text + 16px icon + label (§5.3). Never colour-only; never a bare dot.

### 10.10 Table

For the inspector, webhooks, batches and settings lists. Header row on neutral-50 with `subheading` labels; 52px rows; row hover neutral-50; sortable headers with `aria-sort`; numbers right-aligned and tabular; method and status as chips; paths in `code`. Virtualised beyond 100 rows. Empty and loading states per §10.13.

### 10.11 Batch field

A batch shows counts per status (tiles, tabular) above a **status-cell field**: one 6px cell per recipient, 2px gaps, coloured and patterned by status (delivered solid, undelivered diagonal hatch, failed cross-hatch, queued outline), drawn on canvas for 10,000 cells. Hover or keyboard focus on a cell shows the recipient and status; a table view toggle gives the accessible alternative.

### 10.12 Sandbox pill

Full pill with a 1px **dashed** ember-600 border, ember-700 text, `ShieldCheck` icon: "Sandbox: nothing is delivered". Clicking opens a popover explaining what mocksms does and does not do. Always visible on every app screen; also on the docs site header.

### 10.13 Empty, loading and error states

- **Empty:** a 48px icon tile, a `heading`, one helpful sentence, and one primary action. The first-run inbox shows the get-started checklist (§10.14) instead.
- **Loading:** skeletons shaped like the final rows (no spinners for lists); spinners only inside buttons.
- **Error:** inline where it happened, naming the problem and the recovery; toasts only for transient results.
- **Disconnected:** if the live connection to the server drops, a warning banner under the top bar: "Lost connection to mocksms. Reconnecting…" with retry; it clears itself on reconnect.

### 10.14 Get-started checklist

From the reference's onboarding card, adapted: "Get started with mocksms", a progress label ("2 of 4 done") and four steps that tick themselves off as events arrive:

1. Send your first email to `localhost:1025`
2. Send an SMS with the native API (copyable `curl`)
3. Point a provider SDK at mocksms (Twilio and Termii guides)
4. Read an OTP from a test (`GET /api/v1/otp/latest`)

Each step has a copy tile for its command. Dismissible once complete; reachable later from Settings.

### 10.15 Dialogs, sheets, toasts

Dialogs only for destructive confirmation or focused tasks (credential linking); `Esc` and an explicit Cancel always close them. On widths under 768px, the message detail opens as a full-height sheet. Toasts: bottom right, level 2, auto-dismiss after 4s, `aria-live="polite"`, never steal focus.

---

## 11. Surfaces

### 11.1 Inbox (Operate)

**First viewport (desktop, 1440):** sidebar (248) · top bar (64) · page header with segmented tabs (All, SMS, Email, OTP), search and actions ("Send test message" primary, "Clear" ghost) · list pane (≈420px card) · detail pane (remaining width card). Detail shows recipient header, the large code tile, the delivery track, then the conversation (SMS bubbles: outbound white with neutral-200 border on the right; inbound teal-tinted on the left; reply box at the bottom from M3) or the email plate.

**Screens and their leading content:**

| Screen | Leads with |
|---|---|
| Inbox | Message list and detail; get-started checklist when empty |
| OTPs | Grid of large code tiles, newest first, with verification status |
| Batches | Batch list; batch detail with count tiles and the status-cell field |
| Inspector: Requests | Request table; detail split into raw request and raw response code blocks |
| Inspector: Webhooks | Delivery table with attempts and "Replay" |
| Estimate | Volume inputs, monthly estimate per provider, "Prices as of" date |
| Settings | Project name, linked credentials, webhook URLs, sender allow-list, failure simulation, theme, shortcuts |

**Responsive:** ≥1024 full shell; 768 to 1023 sidebar collapses to the rail and detail opens as a right sheet over the list; <768 sidebar becomes a top-left menu drawer, list and detail become separate views with a back button, inputs and body go to 16px, targets to 44px.

### 11.2 Documentation site (Read)

The world owns the frame; the reading column stays calm.

- **Frame:** white header with wordmark, search and the Sandbox pill; left navigation tree (guides, providers, API reference); right "On this page" contents; ember active states exactly as in the app sidebar.
- **Column:** Figtree `body-lg` 16/26, 68-character measure, neutral-900 text, headings in the app's scale, more space above headings than below.
- **Code:** dark blocks (neutral-950 ground, neutral-50 text, ember-400 for highlighted tokens) with language tabs (Node, Python, PHP, Go, C#) as segmented tabs and a copy button, echoing the reference's dark API block.
- **Signature in docs:** configuration values (ports, env vars, base URLs) appear as copyable code tiles, the same component as in the app.
- **Callouts:** tinted ground (info, warning, success) with icon and a full 1px border, never a thick left bar.
- **Wayfinding:** breadcrumb, previous and next links, deep links on every heading.

### 11.3 Landing page (Persuade)

Inherits this world at full commitment; gets its own impeccable surface round (`concept-seed --scope surface --mode persuade`) inside the pinned world when it is built.

- **Opening:** a headline of at most two lines and a subline of at most 20 words, for example "Test every SMS, OTP and email. Pay nothing until launch." / "mocksms fakes Twilio, Termii and SMTP on your machine, so your app sends for real and nothing leaves your laptop." The primary action is in its working form: the install command for the visitor's platform (tabs: Docker, Homebrew, Scoop, binary) in a copy tile.
- **Proof, not claims:** beside the opening, a live demo built from the real inbox components (not a screenshot made of divs): a scripted message arrives every few seconds with its code tile and delivery track, and a "Send a test OTP" button triggers one. Labelled as a demo.
- **Sections:** an ember statement band (white large text on ember-600, ≥24px); how it works shown with the real two-line config change per provider; providers supported (names as text until X-02); "Free until production" with a go-live estimate preview instead of pricing; "Build with the API" in a dark code block; a closing ember band repeating the install command.
- **Claim and proof:** every claim sits next to the interface detail that proves it.
- **Rules:** no labels above headings, no section numbers, one action intent on the page ("Install"), no customer logos, testimonials or invented statistics, light theme locked for the whole page.

---

## 12. Content and voice

### 12.1 Voice

Warm and encouraging: friendly and supportive, never cute. Errors are specific and say how to recover. Sentence case everywhere. No exclamation marks in errors or warnings. **No em dashes or en dashes in any UI copy** (use periods, commas, colons or parentheses).

### 12.2 Examples

| Moment | Copy |
|---|---|
| Empty inbox | "Your inbox is ready. Send a message to localhost:1025 or POST /api/v1/sms and it will show up here instantly." |
| No OTPs yet | "No codes yet. When your app sends a verification code, it will appear here, ready to copy." |
| Filter with no results | "Nothing matches these filters. Try a wider time range or clear the search." |
| Copied | "Copied" |
| Simulated failure | "Failed (simulated): +15005550002 is a test number that cannot be routed. Twilio error 21612." |
| Webhook failing | "Your webhook at localhost:3000/hooks returned 500. We'll retry in 30 seconds. Replay it now?" |
| Disconnected | "Lost connection to mocksms. Reconnecting…" then "Can't reach mocksms. It may have stopped. Run mocksms again and this page will reconnect on its own." |
| Exposed host warning | "mocksms is reachable from other devices on your network. Set ui_auth to protect the inbox." |

### 12.3 Formatting

Phone numbers in international format with spaces (`+233 24 123 4567`); codes grouped visually in threes; times relative with absolute tooltips; durations in ms below 1s; money with currency code and the "as of" date; IDs in JetBrains Mono, truncated in the middle (`SM8f3a…c21e`) with full value on hover and copy.

**Landing-page copy** also passes taste's pre-flight: hero within 20 words, no duplicate call-to-action intent, no filler verbs, every visible string re-read before ship.

---

## 13. Quality floor

From impeccable's craft floor, applied to every surface:

- **Contrast** body and placeholder ≥4.5:1, large text ≥3:1, UI components and focus ≥3:1 (values in §5.4).
- **Browser surfaces are themed:** `::selection` ember-200 / neutral-900 (dark: ember-800 / ember-50); `caret-color` ember-600; thin scrollbars in neutral-300 (dark 700); `text-underline-offset: 3px`; tabular numerals in data; focus rings from the palette.
- **States** for every interactive element: hover, focus-visible, active, disabled, loading, error, empty.
- **Refused by default:** labels above headings; section numbers; gradient text; glass or blur as decoration; coloured side borders thicker than 1px on cards, rows or callouts; hard offset shadows; monospace as costume; emoji icons; equal icon-heading-text card rows as page structure; modals for tasks that need no interruption.
- **Copy** names actions and errors name the recovery.

---

## 14. Accessibility (WCAG 2.2 AA)

- **Keyboard:** everything reachable and operable; visible focus (2px ember-600 ring, 2px offset; never removed); skip link to main content; logical tab order; list shortcuts (`j`/`k`, `Enter`, `c`, `/`) can be turned off in Settings.
- **Targets:** at least 24×24 CSS px on desktop (2.5.8), 44×44 on touch layouts.
- **Status and colour:** icon + label always; batch cells use patterns as well as colours.
- **Live updates:** new messages announced through a polite live region, batched ("3 new messages") at most every 5 seconds; never steal focus.
- **Structure:** landmarks (`nav`, `main`, `aside`), one `h1` per view, sequential headings, labelled icon-only buttons, form labels above inputs with errors below and linked via `aria-describedby`.
- **Motion:** honours `prefers-reduced-motion` (§9).
- **Zoom and reflow:** usable at 200% zoom and 320px width without horizontal scrolling (except code blocks, which scroll inside themselves).
- **Pre-delivery checklist** (ui-ux-pro-max): contrast in both themes, focus order, labels, reduced motion, 375px and landscape, dark mode checked separately.

---

## 15. Implementation

### 15.1 Tokens (Tailwind v4)

```css
@import "tailwindcss";

@custom-variant dark (&:where([data-theme="dark"], [data-theme="dark"] *));

@theme {
  --font-sans: "Figtree Variable", ui-sans-serif, system-ui, sans-serif;
  --font-mono: "JetBrains Mono Variable", ui-monospace, monospace;

  --color-ember-50: #FFF6ED;  --color-ember-100: #FFEAD3; --color-ember-200: #FED2A6;
  --color-ember-300: #FDB26E; --color-ember-400: #FB8C3A; --color-ember-500: #F76E14;
  --color-ember-600: #E05409; --color-ember-700: #BA3F0A; --color-ember-800: #943311;
  --color-ember-900: #772D11; --color-ember-950: #401406;

  --color-neutral-50: #F7F8FA;  --color-neutral-100: #EDF0F3; --color-neutral-200: #DFE3E8;
  --color-neutral-300: #C6CDD5; --color-neutral-400: #9BA5B1; --color-neutral-500: #687382;
  --color-neutral-600: #535D6B; --color-neutral-700: #3C4552; --color-neutral-800: #272E38;
  --color-neutral-900: #181D24; --color-neutral-950: #0F1318;

  --radius-control: 8px;
  --radius-container: 12px;

  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --ease-in: cubic-bezier(0.7, 0, 0.84, 0);
}
```

The theme is applied once at the root: `data-theme` is set from the saved preference or `prefers-color-scheme`. Components use semantic variables, never raw ramp values.

### 15.2 shadcn semantic mapping

| shadcn variable | Light | Dark |
|---|---|---|
| `--background` | neutral-50 | neutral-950 |
| `--card`, `--popover` | white | neutral-900 / neutral-800 |
| `--foreground` | neutral-900 | neutral-50 |
| `--muted` / `--muted-foreground` | neutral-100 / neutral-600 | neutral-800 / neutral-300 |
| `--primary` / `--primary-foreground` | ember-500 / ember-950 | ember-500 / ember-950 |
| `--accent` / `--accent-foreground` | ember-50 / ember-700 | ember-800 at 30% / ember-400 |
| `--border`, `--input` | neutral-200 | neutral-800 |
| `--ring` | ember-600 | ember-400 |
| `--destructive` | `#B91C1C` | `#F87171` |

### 15.3 Dependencies added by this design

`@phosphor-icons/react`, `@fontsource-variable/figtree`, `@fontsource-variable/jetbrains-mono` (recorded in [techstack.md](techstack.md)). No animation library: CSS transitions cover §9.

---

## 16. Workflow: using the design skills during the build

| When | Do |
|---|---|
| Starting a surface (M1-17, landing, docs) | Record the direction contract (§17) with `impeccable surface-brief write`; for the landing page and docs site, run their impeccable surface round inside this world |
| Before any UI edit | Read impeccable's `reference/craft-floor.md` |
| While building | Follow this file; ui-ux-pro-max checklists for accessibility, interaction and forms |
| After a UI task | `impeccable detect --json <changed files>` (no hook is installed), fix mechanical findings |
| End of M1 (and each UI milestone) | `/impeccable critique` and `/impeccable audit`; the impeccable finish reviewer on desktop and mobile screenshots; then the impeccable documenter writes root `DESIGN.md` and `.impeccable/design.json` from the built inbox |
| Copy passes | design:ux-copy against §12 |
| After M1 | design:design-system audit of the shipped components; design:accessibility-review |
| Landing page | taste's full pre-flight checklist plus impeccable's Persuade rules |

---

## 17. Direction contract (inbox)

To be recorded in impeccable's surface brief at the start of M1-17. Development-only; never copied into source, comments or anything served to the browser.

- **THESIS:** A warm, light sandbox inbox where the code is the hero; it refuses the grey admin template and the hacker terminal alike.
- **OWN-WORLD:** Cool neutral ramp, white 12px surfaces, one ember-orange accent doing real jobs, Figtree for the interface, JetBrains Mono for codes, duotone Phosphor icons, a dashed Sandbox pill.
- **STORY:** The developer triggers a send, sees the message arrive within a second, copies the code, and trusts that it behaved like the real provider, failures included.
- **FIRST VIEWPORT:** Sidebar left; top bar with counters and the Sandbox pill; segmented tabs and "Send test message" top right; list pane with the newest message lit; detail pane leading with the large code tile and the delivery track.
- **FORM:** Owner-pinned reference (warm orange bulk-SMS dashboard), superseding direction rounds 116a3885 and its re-roll.
- **FINISH:** unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

---

## 18. Open design items

| Item | When |
|---|---|
| Logo mark (dashed speech bubble concept) via the logo-design skill | Before the landing page ships |
| Provider logos in docs and landing | After the legal check (task X-02) |
| Landing and docs surface rounds inside this world | When those surfaces are scheduled |
| Confirm Figtree `tnum` support | M1-17 |
