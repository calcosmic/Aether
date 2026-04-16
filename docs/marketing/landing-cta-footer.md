# Landing Page: CTA, Footer, and Social Proof

> Structured markdown for web developer implementation.
> Tone: confident, clear, non-technical for CTAs and social proof.
> All social proof items are placeholders until real data is available.

---

## 1. Primary CTA Section (Hero / Bottom of Page)

**Layout note:** Full-width section with centered content. Dark or accent background to contrast with the main body. Should appear both below the hero and as a sticky/final section before the footer.

### Headline

> Ship software while you sleep.

### Subtext

> Aether deploys 24 specialized workers that self-organize around your goal. No prompt engineering, no babysitting. Set it up, set a goal, and let the colony build.

### Primary CTA Button

- **Label:** `Get Started -- It's Free`
- **Link:** `https://github.com/calcosmic/Aether`
- **Style:** Bold, high-contrast (e.g., white text on purple/brand background). Large touch target. Slight hover animation (scale or glow).

### Secondary CTA

- **Label:** `Read the Quick Start`
- **Link:** `https://github.com/calcosmic/Aether#install`
- **Style:** Outlined/ghost button. Smaller, sits beside or below the primary CTA.

### Install Command Block

```
Layout note: Monospace font, dark code block with a "copy" button.
Position: Directly below the CTA buttons, centered.
Pre-label: "Install in one line:"
```

```bash
go install github.com/calcosmic/Aether@latest
```

### Supplementary Install Options

```
Layout note: Small text below the install block, gray/muted color.
```

> No Go? Download pre-built binaries for macOS, Linux, and Windows from [GitHub Releases](https://github.com/calcosmic/Aether/releases).

---

## 2. Secondary CTA (Mid-Page)

**Layout note:** Positioned after the features/comparison section, before social proof. Light background, minimal. Purpose: capture visitors who are interested but not ready to install.

### Headline

> Not ready to install? Start with the docs.

### Body

> See how the colony works before you commit. The README walks through a full build -- from blank directory to shipped API -- so you can decide if Aether fits your workflow.

### CTA Button

- **Label:** `Explore the Docs`
- **Link:** `https://github.com/calcosmic/Aether#readme`
- **Style:** Outlined button, brand color border. Moderate size.

### Optional: Newsletter Signup

```
Layout note: Inline form below the CTA button. Email input + "Get updates" button.
Purpose: Capture interested visitors for launch updates.
```

- **Input placeholder:** `you@email.com`
- **Button label:** `Get Updates`
- **Privacy note (below form):** "No spam. Just updates when something ships."

---

## 3. Footer

**Layout note:** Dark background (#1a1a2e or similar), full width. Multi-column layout on desktop, stacked on mobile.

### Footer Columns

**Column 1: Brand**

> **Aether**
> The whole is greater than the sum of its ants.
>
> Open-source multi-agent orchestration modeled on ant colonies. Built with Go. Apache 2.0 licensed.

**Column 2: Project**

- [GitHub](https://github.com/calcosmic/Aether)
- [Website](https://aetherantcolony.com)
- [Documentation](https://github.com/calcosmic/Aether#readme)
- [Quick Start](https://github.com/calcosmic/Aether#install)
- [GitHub Releases](https://github.com/calcosmic/Aether/releases)

**Column 3: Community**

- [Discord] `[PLACEHOLDER -- insert Discord invite link]`
- [GitHub Discussions](https://github.com/calcosmic/Aether/discussions)
- [Twitter / X] `[PLACEHOLDER -- insert Twitter/X profile link]`
- [Contributing Guide](https://github.com/calcosmic/Aether#contributing)

**Column 4: Legal**

- [Apache 2.0 License](https://github.com/calcosmic/Aether/blob/main/LICENSE)
- [Code of Conduct] `[PLACEHOLDER -- insert CoC link when added]`
- [Security Policy] `[PLACEHOLDER -- insert SECURITY.md link when added]`

### Footer Badges Row

```
Layout note: Horizontal row of badges, centered, above the copyright line.
Use shields.io badges for visual consistency.
```

- ![Apache 2.0 License](https://img.shields.io/github/license/calcosmic/Aether.svg?style=flat-square)
- ![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)
- ![Latest Release](https://img.shields.io/github/v/release/calcosmic/Aether.svg?style=flat-square)
- ![Built with Go](https://img.shields.io/badge/Built%20with-Go-00ADD8?style=flat-square&logo=go)

### Copyright Line

> Copyright 2026 Cal Cosmic. Released under the [Apache 2.0 License](https://github.com/calcosmic/Aether/blob/main/LICENSE).

---

## 4. Social Proof Section

**Layout note:** Positioned between the secondary CTA and the primary CTA (or above the primary CTA). Light/white background. All items are placeholders until real data exists.

> **[PLACEHOLDER]** -- This entire section should be hidden or visually de-emphasized until real social proof is available. Consider using a muted gray style or a "coming soon" label.

### 4a. Testimonials

```
Layout note: 3-column card grid on desktop, stacked on mobile.
Each card: Quote text (italic), author name, author title/org, avatar placeholder.
```

**[PLACEHOLDER] Testimonial 1:**

> "Aether changed how I approach building features. I describe what I want, and the colony figures out the implementation. The pheromone system means I never have to micromanage prompts."

-- **[PLACEHOLDER Name]**, [PLACEHOLDER Title] at [PLACEHOLDER Company]

**[PLACEHOLDER] Testimonial 2:**

> "We went from zero to a working API in a single session. The autopilot mode is unreal -- it paused when it hit something it couldn't solve, we gave it a nudge, and it finished the rest."

-- **[PLACEHOLDER Name]**, [PLACEHOLDER Title] at [PLACEHOLDER Company]

**[PLACEHOLDER] Testimonial 3:**

> "The fact that learnings carry between projects is what sold me. The colony remembers what worked last time and applies it without being asked. It genuinely gets smarter the more you use it."

-- **[PLACEHOLDER Name]**, [PLACEHOLDER Title] at [PLACEHOLDER Company]

### 4b. GitHub Stars Counter

```
Layout note: Centered badge or dynamic counter widget.
Use the GitHub API or shields.io for a live count.
```

![GitHub Stars](https://img.shields.io/github/stars/calcosmic/Aether.svg?style=social&label=Star)

> **[PLACEHOLDER]** -- Star count will update automatically via shields.io. No manual maintenance needed.

### 4c. "Trusted By" / Logo Wall

```
Layout note: Horizontal row of grayscale logos, centered.
Use max-height to keep logos uniform size.
Label above: "Used by developers at" or "Trusted by"
```

> **[PLACEHOLDER]** -- Add company/organization logos here as users adopt Aether. Recommended format: SVG logos at 120px width, grayscale filter. Minimum 3 logos before making this section visible.

**[PLACEHOLDER]** -- Logo slot 1
**[PLACEHOLDER]** -- Logo slot 2
**[PLACEHOLDER]** -- Logo slot 3
**[PLACEHOLDER]** -- Logo slot 4
**[PLACEHOLDER]** -- Logo slot 5

### 4d. Numeric Highlights (Optional)

```
Layout note: Row of 3-4 stat cards with large numbers and labels.
Position: Above testimonials, below the "Trusted by" section.
```

| Stat | Value | Label |
|------|-------|-------|
| **[PLACEHOLDER]** | `24` | Specialized workers |
| **[PLACEHOLDER]** | `28` | Built-in skills |
| **[PLACEHOLDER]** | `45` | Slash commands |
| **[PLACEHOLDER]** | `1,000+` | GitHub stars (update dynamically) |

> **Note:** The worker, skill, and command counts are real and can be displayed immediately. Only the GitHub stars value is a placeholder.

---

## 5. SEO Meta Description

```
Length: 153 characters (under 160 limit)
```

> Aether is an open-source multi-agent orchestration tool where 24 specialized AI workers self-organize around your goals. Built in Go, Apache 2.0 licensed.

**HTML meta tag:**

```html
<meta name="description" content="Aether is an open-source multi-agent orchestration tool where 24 specialized AI workers self-organize around your goals. Built in Go, Apache 2.0 licensed.">
```

**Open Graph description (optional, same text):**

```html
<meta property="og:description" content="Aether is an open-source multi-agent orchestration tool where 24 specialized AI workers self-organize around your goals. Built in Go, Apache 2.0 licensed.">
```

**Title tag suggestion:**

```html
<title>Aether -- Multi-Agent AI Orchestration | Open Source, Built with Go</title>
```

---

## 6. Developer Implementation Notes

### Page Section Order (recommended)

1. Hero (headline + primary CTA)
2. Features / How It Works
3. Comparison Table (Aether vs Others)
4. Secondary CTA (mid-page)
5. Social Proof (testimonials, stars, stats)
6. Primary CTA (final push)
7. Footer

### Color Palette Guidance

- **Primary brand:** Purple (#7B3FE4, used in shields.io badges)
- **CTA background:** Purple or brand accent
- **Secondary CTA:** Outlined, same brand color
- **Footer background:** Dark (#1a1a2e or similar)
- **Social proof background:** Light gray or white
- **Code block background:** Dark (#0d1117 or similar, GitHub style)

### Responsive Behavior

- **Desktop (1024px+):** Multi-column footer, 3-column testimonial grid, side-by-side CTA buttons
- **Tablet (768px-1023px):** 2-column footer, 2-column testimonials, stacked CTAs
- **Mobile (<768px):** Single column everything, full-width buttons, stacked footer sections

### Placeholder Removal Checklist

When real data becomes available, replace these items:

- [ ] All `[PLACEHOLDER]` testimonial entries with real quotes
- [ ] `[PLACEHOLDER]` social links (Discord, Twitter/X) with real URLs
- [ ] `[PLACEHOLDER]` Code of Conduct link with actual page
- [ ] `[PLACEHOLDER]` Security Policy link with actual page
- [ ] `[PLACEHOLDER]` "Trusted by" logo wall with real logos
- [ ] `[PLACEHOLDER]` GitHub stars count (or use shields.io auto-update)
- [ ] Remove the placeholder notice above the social proof section once 2+ real testimonials exist
