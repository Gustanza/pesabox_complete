# DESIGN.md — the Finwave design language

Reference: **Finwave** WordPress theme by RadiusTheme, *Home 03* demo
(<https://www.radiustheme.com/demo/wordpress/themes/finwave/home-03/>).
Studied from the live HTML and CSS (`style.css`, the Elementor page CSS and
the theme's `:root` variables) on 2026-09-28. Values below are the theme's
own, not estimates, unless marked *(approx.)*.

This file describes **how that design works** so it can be reproduced in our
own code (landing page, web dashboard, app screens). It is a spec, not a copy:
no Finwave assets, copy, or code should be shipped — only the language.

---

## 1. The idea in one paragraph

A **confident fintech look built on one deep green**. Big, heavy, tightly
set Archivo headlines; a single warm **amber** accent used sparingly for
emphasis and hover; lots of white and pale-mint space between full-bleed
green "statement" bands. Surfaces are **soft-cornered (20–30 px)** but
carry a **hard, offset, un-blurred shadow** (`10px 10px 0`) that gives a
playful, slightly neo-brutalist edge to an otherwise calm, trustworthy
palette. Product is shown through **phone mockups and floating UI chips**,
people through **cut-out lifestyle photos**, and decoration through
**blurred colour blobs, concentric rings, dot grids, noise texture and
amber arc shapes**. Motion is gentle: fade/slide-up on scroll, fill-wipe
buttons, a rotating word in the hero.

Keywords: *trustworthy · bold · friendly · green-and-amber · soft shapes,
hard shadows · lots of air.*

---

## 2. Colour

### 2.1 Core tokens (from the theme's `:root`)

| Token | Hex | RGB | Role |
|---|---|---|---|
| `--primary` | `#006D5B` | 0,109,91 | Brand green. Statement bands, primary buttons, eyebrows, icons, links on hover, selection. |
| `--secondary` | `#00473B` | 0,71,59 | Deep green. Footer, button hover fill, gradient end. |
| `--tertiary` (accent) | `#FFB000` | 255,176,0 | Amber. Hover states, highlights, active nav item, star ratings, decorative arcs, CTA on dark. |
| `--title` | `#041B16` | 4,27,22 | Headings and strong text — a green-black, never pure black. |
| `--body` | `#6E7676` | 110,118,118 | Paragraph text — a grey with a green cast. |
| `--border` | `#E7E7E7` | | Card and input borders, dividers. |
| `--bg` | `#FFFFFF` | | Page background. |
| `--gray10` | `#F1F1F1` | | Neutral fill (code, kbd, quiet panels). |
| `--gray20` / mint | `#EDF5F4` | | **Tinted surface** — testimonial cards, soft panels, inactive stars. |
| `--meta-light` | `#B2C2C0` | | Muted text on green (eyebrows on dark). |
| `--button-text` | `#00030C` | | Text on amber buttons. |

### 2.2 Supporting values seen in use

| Hex | Where |
|---|---|
| `#FFA700` | Amber variant used for the **hard hover shadow** on cards. |
| `#E6EBEB` / `#E2E2E2` | Body/description text on green and in the footer. |
| `#177D6C` | Lighter green — header CTA button fill. |
| `#0D8571`, `#2F614F` | Mid greens in gradients/illustrations. |
| `#255A51`, `#28534F` | Hairline borders on green (footer divider, header bottom line). |
| `#D0E2DF` | Mint highlighter bar under some titles. |
| `rgba(0,0,0,.30)` | The default **hard shadow** colour. |

### 2.3 Decorative blob colours (blurred, 50 % alpha)

`rgba(186,241,130,.5)` lime · `rgba(226,130,241,.5)` orchid ·
`rgba(130,135,241,.5)` periwinkle · `rgba(130,241,208,.5)` aqua ·
`#EDFBF3` / `#EAE8FF` / `#E5E6FF` pale wash blobs.
They sit behind content with `filter: blur(100–150px)` — felt, not seen.

### 2.4 How colour is distributed

- **~70 % white / mint**, **~20 % brand green** (hero, one mid-page band,
  CTA band, footer), **≤5 % amber**, the rest title/body greys.
- Page rhythm alternates **light section → green band → light section**.
  Never two green bands back-to-back except CTA → footer (which use two
  different greens: `#006D5B` then `#00473B`).
- On green, text is white (`#FFFFFF`) for titles and `#E6EBEB` for body;
  eyebrows go `#B2C2C0` or amber.
- Amber is **the only warm colour**. It marks *interaction* (hover, active,
  focus) and *one* highlight per view (a word, an arc shape, stars). Don't
  use it for large fills except a single CTA button on a green band.
- Text selection: green background, white text.

---

## 3. Typography

**One family: Archivo** (Google Fonts, weights 100–900). Headings, body,
menu and buttons all use it. Personality comes from **weight contrast**, not
from mixing fonts.

| Style | Size / line-height | Weight | Colour | Notes |
|---|---|---|---|---|
| Hero H1 | **60 px** / 1.2 | **900** | white on green | Can contain an animated rotating phrase. |
| Section title (H2) | **40 px** default, **36 px** on this page / 1.2 | 700 | `--title` | A `<span>` inside drops to **400** for a light/heavy contrast. |
| Stat number | **80–100 px** hero-size, **48 px** in grids / 1.2 | 700 | title or white | Followed by a smaller unit (`K`, `%`, `+`, `M`). |
| H1 (content pages) | 44 / 54 px | 700 | `--title` | |
| Card title (H3) | 22 px / 1.3–1.4 | 700 | `--title` | |
| Footer column title | 20 px | 700 | white | |
| **Eyebrow** (top sub-title) | **14 px** / 25 px, **UPPERCASE**, letter-spacing **0.96 px** | 700 | `--primary` (or `#B2C2C0`/amber on dark) | Sits above every section title. |
| Lead / description | 18 px / 1.7 | 500 | `--body` (or `#E6EBEB` on green) | |
| Body | **16 px / 28 px** | **500** | `--body` | Medium weight body is part of the look — not 400. |
| Nav link | 16 / 22 px | 600 | `--title` (white on transparent header) | Active item = amber. |
| Button | 16 px, line-height 1 | 700 | — | letter-spacing 0.5 px. |
| Testimonial quote | 17 px | 500 | `--body` | |
| Footer phone number | 22 px | 700 | white | Contact details as a display element. |

Rules:
- Headlines are **Title Case**, short, benefit-led ("Finding A Solution
  That's Perfect For You").
- Every section title is preceded by an **eyebrow** (e.g. `FEATURES THAT
  YOU WILL LOVE`, `WHY WE ARE THE BEST`, `OUR TESTIMONIALS`).
- Mobile (≤ 767 px): hero/title drop to **36 px**, stat numbers to **48 px**.
- Headings default margin: `0 0 15px`.

---

## 4. Shape, depth & texture

### 4.1 Radius scale

| Radius | Used for |
|---|---|
| **8 px** | Buttons, inputs, small chips. |
| **12 px** | Small floating images / UI chips. |
| **20 px** | Cards (info boxes, testimonial cards). |
| **24–30 px** | Large image frames, feature panels. Also half-rounded variants: `24px 0 0 24px` (image bleeding off the right edge) and `45px 45px 0 0` (phone-screen "tab" frame). |
| 50 % | Avatars, back-to-top button, icon discs. |

### 4.2 Shadows — the signature

| Name | Value | Use |
|---|---|---|
| **Hard offset** | `10px 10px 0 0 rgba(0,0,0,.3)` | Feature cards at rest (on green bands). |
| **Hard offset — amber** | `10px 10px 0 0 #FFA700` | Same card **on hover** (shadow turns amber). |
| **Hard offset — small** | `-5px 5px 0 0 rgba(0,0,0,.4)` + `1px solid #444` border | Small floating UI chips over photos. |
| Soft header | `6px 6px 32px 0 rgba(0,80,191,.08)` | Sticky header once scrolled. |
| Soft float | `0 4px 50px rgba(0,0,0,.15)` | Back-to-top button. |

The mix is deliberate: **soft blur shadows only for chrome** (header,
floating button); **hard, zero-blur offsets for content cards**. That hard
shadow is the single most recognisable detail — keep it.

### 4.3 Borders

- Cards: `1px solid var(--border)` on white; on hover the border turns
  `--primary`.
- Phone/device frames: thick black borders (`10px 10px 0 10px #000`)
  drawn in CSS around a screenshot, top-rounded 45 px.
- On green: hairlines in `#255A51`.

### 4.4 Texture & decoration

- **Noise overlay** on green bands (a grain PNG, `background-size: cover`)
  — keeps large green areas from looking flat.
- **Dot grid** SVG on the green stats band, `background-attachment: fixed`
  (parallax feel).
- **Concentric white rings** on pale-mint panels (`#EDF5F4`-ish) behind
  product shots.
- **Amber arc shapes** (thick quarter-ring strokes, `#FFB000`) peeking from
  behind images.
- **Dotted circle** outline behind the hero phone.
- **Blurred colour blobs** (see 2.3) behind stats and sections.
- Small **spinning** badge/shape (`animation: spin 15s linear infinite`)
  next to titles.
- Eyebrow variant: a **ribbon** — green→deep-green gradient (`111deg`),
  white text, notched ends via
  `clip-path: polygon(0 0,100% 0,95% 50%,100% 100%,0 100%,5% 50%)`.

---

## 5. Layout & spacing

- **Container: 1364 px** max width. Header uses a fluid container with
  **70 px** side padding.
- 12-column grid (Bootstrap-style `col-xl-6 col-lg-6 col-md-6`), gutter
  `g-4` (24 px).
- Section vertical padding *(approx.)*: **100–120 px** desktop, ~60–80 px
  tablet, ~50–60 px mobile. Green bands get the same or more.
- Card padding: **35 × 45 px** (info box), **40 px** (testimonial),
  **30–60 px** (feature panels).
- Section heading block: eyebrow → 7 px → title → 15 px → description →
  ~40–60 px → content.
- Common compositions:
  - **Split 50/50**: text + CTA on one side, framed image/phone + floating
    chips on the other. Sides alternate down the page.
  - **3-up** or **4-up** card rows for features.
  - **Centred heading** above grids, **left-aligned heading + button on the
    right** above sliders ("title left, action right").
  - Images **bleed** to the container edge (half-rounded corners on the
    inside edge only).
- Breakpoints used: ≤ 767 px (mobile), ≤ 1024 px (tablet); sliders go to 1
  slide per view at 0.

---

## 6. Components

### 6.1 Header
- **Transparent over the green hero** ("light" header — white logo, white
  nav), absolute-positioned. On scroll it becomes **sticky, white, soft
  shadow**, dark logo (headroom behaviour: hides on scroll down, shows on
  scroll up).
- Left: logo. Centre: nav (16/600, 26 px × 15 px link padding, mega-menus).
  Right: search icon, **"Login"** text-button (white → amber on hover), and
  a **filled CTA button** (`#177D6C` → amber wipe, text turns `#00030C`).
- Thin bottom hairline `#28534F` on the transparent state.
- Mobile: hamburger → **off-canvas drawer** (logo, short about, nav,
  address, socials).
- Tiny nav badges: `new` (blue), `hot` (pink bg, red text), `sale` (lime).

### 6.2 Buttons
Base: `display:inline-flex; gap:10px; padding:16px 26px; height:55px;
border-radius:8px; font:700 16px/1 Archivo; letter-spacing:.5px;
overflow:hidden;` with an arrow icon (12–14 px) on the right.

**The fill-wipe hover** (the theme's button signature): two stacked pseudo
layers — `::before` is the resting colour, `::after` sits at
`translateY(-100%)` and **drops down** on hover with a springy
`cubic-bezier(0.52, 1.64, 0.37, 0.66)` over 0.5 s. The arrow icon nudges
`translateX(5px)`.

| Variant | Rest | Hover wipe | Text |
|---|---|---|---|
| **Primary** (button-1) | green `#006D5B` | deep green `#00473B` | white |
| **Accent** (button-2) | amber `#FFB000` | green `#006D5B` | `#00030C` → white |
| **Outline** (button-4) | white + 1 px `#E7E7E7` border | green | `#00030C` → white |
| **On green band** | green/amber | **white** | white → `#00030C` |
| Text/login | transparent, no padding | — | white → amber |

Sizes: `xs 11×18/10px`, `sm 12×20/13px`, default `16×26/16px`,
`lg 18×24/18px`, `xl 24×32/24px`.

### 6.3 Section title block
Eyebrow (14 px uppercase green, optional icon or ribbon) → main title
(36–40 px bold, optional light-weight span, optional rotating/gradient
animated word, optional hand-drawn SVG underline that draws in) →
description (18 px / 1.7). Alignment: centred for grids, left for splits.

### 6.4 Info box (feature card)
White, `1px solid #E7E7E7`, **radius 20 px**, padding **35 × 45 px**,
overflow hidden. Icon (40 px, green) → 24 px gap → title (22 px bold) →
text. Hover: border → green, icon **scales 1.1 and turns amber**, card
gets/keeps the **hard offset shadow** (black 30 % → amber on hover).
Transitions 520 ms ease. Variant with a small image in place of the icon.

### 6.5 Counter / stat
Huge number (80–100 px, or 48 px in 4-up grids) + unit + label below
(16 px). On green: white numbers. Behind each: a **blurred colour blob**
(160 px circle, `blur(125px)`, one colour per stat). Numbers count up on
scroll. Outline variant: transparent fill with `-webkit-text-stroke: 2px`
that fills in on hover.

### 6.6 Testimonial card
Mint `#EDF5F4` background, **radius 20 px**, padding **40 px**. Order:
star rating (22 px, amber active / mint inactive) → quote (17 px) → author
row (round avatar + name 22 px bold + role). Shown in a slider, usually 2
visible on desktop, 1 on mobile, next to a left-aligned title + "view all"
button.

### 6.7 Logo strip
Brand logos in a row/slider, **greyscale at 50 % opacity**, full colour
on hover (0.21 s). On dark: `brightness(0) invert(1)` at 40 % → 100 %.

### 6.8 Blog / post card
Image (960×690 ratio ≈ 4:3, rounded) that **zooms to 1.05** on hover →
category + date meta (green) → title 22 px / 1.4 bold. Shown in a slider
with left title + right arrows/button.

### 6.9 Product showcase
- Phone mockups (held in hand, or framed with CSS borders) showing a
  **dark-green balance card** and white rounded list rows.
- **Floating UI chips** around them: small white cards (radius 12 px, hard
  offset shadow, thin dark border) showing a stat, a mini table, a graph.
- Behind: mint ring panel, dotted circle, or amber arc.

### 6.10 CTA band + footer
- CTA: full-width green `#006D5B` with noise; big white title ("Get Started
  Today …"), eyebrow in `#B2C2C0`, a button whose hover wipe is **white**
  (text flips to dark), phone image breaking out of the band.
- Footer: deep green `#00473B`. Columns: logo + contact info (phone number
  22 px bold white) + social icons | Company links | Features links |
  Locations. Column titles 20 px white; links `#E6EBEB` 16/500 → white on
  hover; 8 px vertical link padding. Bottom bar: 1 px `#255A51` top border,
  50 px padding, centred copyright `#E2E2E2`, letter-spacing 1 px.

### 6.11 Small parts
- **Back-to-top**: 40 px green circle, bottom-right 30 px, soft shadow;
  hover → amber with dark icon, arrow loops up.
- **Preloader**: animated brand GIF on white.
- **Search**: full-screen overlay form.

---

## 7. Imagery & iconography

- **Photography**: bright, friendly, studio cut-outs (no background) of
  people holding phones — real, smiling, casual clothing that picks up the
  palette (greens).
- **Product**: phone screens with the app's UI; always a dark-green hero
  card on a mint screen, white rows with rounded icon tiles.
- **Illustration**: flat UI fragments only (tables with flags, check
  badges, small bar graphs) — no characters/cartoons.
- **Icons**: simple line/solid icon font (Fontello + Font Awesome 5),
  single colour (green → amber on hover), 40 px in cards, 12–14 px in
  buttons.

---

## 8. Motion

| What | How |
|---|---|
| Scroll reveal | WOW.js + animate.css: `fadeInUp` / `slideInUp` / `slideInDown`, staggered per item. |
| Buttons | Fill-wipe from top, springy cubic-bezier, 0.5 s; icon nudges 5 px right. |
| Cards | 520 ms ease on border colour, shadow colour, icon scale (1.1). |
| Images | Post images scale 1.05 on hover. |
| Headline | Rotating word ("animated headline"); gradient text animation (`background-position` 0→100→0 %, 5 s loop); SVG underline draws in (`stroke-dasharray`, 1 s). |
| Decoration | Slow spin (15 s linear) on badges/shapes; parallax backgrounds (`background-attachment: fixed`); floating blobs. |
| Counters | Count up when scrolled into view. |
| Header | Headroom: slides away on scroll down, returns on scroll up. |

Motion is always **short, eased, and decorative** — nothing blocks reading.
Respect `prefers-reduced-motion` when we implement it (the theme doesn't).

---

## 9. Voice (copy style)

- Eyebrow: 2–4 words, UPPERCASE, a category or promise
  ("TOP SERVICES", "WHY WE ARE THE BEST").
- Title: Title Case, 5–10 words, benefit-first, sometimes a question or a
  challenge ("Don't Believe Us? People Talk About It").
- Body: one or two short sentences.
- Buttons: verb-led, short ("Get Started", "Learn More", "View All").
- Social proof is explicit and numeric ("200 projects", "150+ clients",
  "10K Daily Active Users").

---

## 10. Page anatomy (Home 03, top to bottom)

1. **Hero** — green `#006D5B` + noise; transparent white header; eyebrow
   "ONLINE BANKING PROCESS"; 60 px/900 white H1 with a rotating phrase;
   description; CTA; two inline stats; right: hand-held phone on a dotted
   circle with an amber arc and floating chips.
2. **Features** — white; centred title; **3 info boxes**.
3. **About / why us** — split: framed photo of a person with a phone, amber
   arc, floating chips | eyebrow "WHY WE ARE THE BEST", title, text,
   button.
4. **Services** — eyebrow "TOP SERVICES", title + button; **4 info boxes**
   (icon + title + text) around/next to an image; on a green band these
   carry the hard black shadow → amber on hover.
5. **Consulting** — split with image panel (mint rings), title, button.
6. **Control your money** — split: phone mockup in a CSS device frame |
   title, text, button.
7. **Case study + stats** — green band with fixed dot pattern; big title;
   **4 counters** (80 px white) with blurred colour blobs.
8. **Testimonials** — left title + button; slider of mint cards with amber
   stars.
9. **Clients** — title "worked on over 200 projects with 150+ clients";
   greyscale logo strip.
10. **Blog** — left title; post slider (image zoom on hover).
11. **CTA band** — green + noise, big white title, white-wipe button, phone
    breaking out.
12. **Footer** — deep green `#00473B`, 4 columns, hairline, centred
    copyright; back-to-top button.

---

## 11. Ready-to-use tokens

```css
:root {
  /* colour */
  --c-primary: #006D5B;   --c-primary-rgb: 0,109,91;
  --c-secondary: #00473B; --c-secondary-rgb: 0,71,59;
  --c-accent: #FFB000;    --c-accent-rgb: 255,176,0;
  --c-accent-shadow: #FFA700;
  --c-title: #041B16;     --c-body: #6E7676;
  --c-border: #E7E7E7;    --c-bg: #FFFFFF;
  --c-mint: #EDF5F4;      --c-gray: #F1F1F1;
  --c-on-dark: #FFFFFF;   --c-on-dark-muted: #E6EBEB; --c-on-dark-meta: #B2C2C0;
  --c-dark-line: #255A51; --c-btn-dark-text: #00030C;

  /* type */
  --font: 'Archivo', system-ui, sans-serif;
  --fs-hero: 60px;  --fs-h2: 40px; --fs-h2-page: 36px; --fs-stat: 80px;
  --fs-h3: 22px;    --fs-lead: 18px; --fs-body: 16px; --lh-body: 28px;
  --fs-eyebrow: 14px; --ls-eyebrow: .96px;

  /* shape */
  --r-btn: 8px; --r-chip: 12px; --r-card: 20px; --r-panel: 30px;
  --shadow-hard: 10px 10px 0 0 rgba(0,0,0,.3);
  --shadow-hard-accent: 10px 10px 0 0 var(--c-accent-shadow);
  --shadow-chip: -5px 5px 0 0 rgba(0,0,0,.4);
  --shadow-header: 6px 6px 32px 0 rgba(0,80,191,.08);

  /* layout & motion */
  --container: 1364px; --gutter: 24px;
  --ease-spring: cubic-bezier(.52,1.64,.37,.66);
  --t-card: 520ms ease; --t-btn: .5s;
}
```

---

## 12. Adapting it to HelaBox *(suggestion — confirm before use)*

The Finwave language fits HelaBox well (green fintech, savings, trust). Our
brand already has its own palette from the logo pack
(`~/uploads/logo/…`, `web/src/assets/logo-icon.svg`):

| HelaBox logo colour | Hex | Maps to Finwave role |
|---|---|---|
| Navy | `#112D4B` | `--title` and/or the **dark band** colour (instead of green), or keep green bands and use navy for text. |
| Teal | `#1598B7` | Could replace `--primary` green, or be a secondary accent. |
| Orange | `#F19135` | Natural stand-in for the **amber accent** `#FFB000`. |
| Purple | `#803E98` | Extra accent (blob colour, badges) — use sparingly. |

Two coherent options:
- **A. Keep Finwave green as primary** (`#006D5B`/`#00473B`), swap amber
  for HelaBox orange `#F19135`, use navy `#112D4B` for titles. Closest to
  the reference; the logo's four colours appear in blobs/illustrations.
- **B. Re-key to HelaBox navy + teal**: bands in navy `#112D4B`, primary
  actions teal `#1598B7`, accent orange `#F19135`. More on-brand, keeps
  every structural idea (hard shadows, eyebrows, fill-wipe buttons, big
  Archivo type, mint-style tinted surfaces → use a pale teal/navy tint).

Whatever the palette, keep the **structure**: Archivo with 500 body / 700–900
headings, eyebrow + title blocks, 20 px cards with the hard offset shadow,
fill-wipe buttons, alternating light/colour bands, phone-mockup product
shots with floating chips.
