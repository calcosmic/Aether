# Visual Asset Specifications for Aether Marketing

**Version:** 1.0.0
**Date:** 2026-04-08
**Author:** Bolt-7 (Builder Ant)
**Status:** Draft for review

---

## Table of Contents

1. [Brand Guidelines](#1-brand-guidelines)
2. [Existing Asset Inventory](#2-existing-asset-inventory)
3. [Missing Assets by Channel](#3-missing-assets-by-channel)
4. [Detailed Asset Specifications](#4-detailed-asset-specifications)
5. [Priority Matrix](#5-priority-matrix)
6. [Production Notes](#6-production-notes)

---

## 1. Brand Guidelines

### 1.1 Color Palette

| Role | Hex | Usage |
|------|-----|-------|
| **Primary Brand** | `#7B3FE4` | Buttons, accents, shields.io badges, links |
| **Primary Dark** | `#5A2DB0` | Hover states, dark accents |
| **Primary Light** | `#A78BFA` | Subtle highlights, secondary elements |
| **Dark Background** | `#0d1117` | Code blocks, terminal styling (GitHub dark) |
| **Footer Background** | `#1a1a2e` | Footer, dark sections |
| **Hero Background** | `#0a0a14` or `#1a1a2e` | Landing page hero section |
| **Text Primary** | `#F9FAFB` | Headings on dark backgrounds |
| **Text Secondary** | `#9CA3AF` | Body text on dark backgrounds |
| **Text Muted** | `#6B7280` | Captions, metadata |
| **Go Blue** | `#00ADD8` | Go language references, secondary accent |
| **Green (Action)** | `#16a34a` | Success states, "build" indicators |
| **Red (Redirect)** | `#EF4444` | Warning states, REDIRECT signal color |

### 1.2 Typography

| Element | Font Family | Weight | Notes |
|---------|-------------|--------|-------|
| **Headlines** | Inter, System UI | Bold (700) | Clean, modern sans-serif |
| **Body Text** | Inter, System UI | Regular (400) | High readability |
| **Code/Mono** | JetBrains Mono, Fira Code, SF Mono | Regular (400) | Developer-facing content |
| **Tagline** | Inter or display serif | Medium (500) | "The whole is greater than the sum of its ants." |

### 1.3 Logo Usage Rules

- **Minimum clear space:** Logo height x 0.5 on all sides
- **Minimum size:** 32px height for digital, 0.5 inch for print
- **Never:** Stretch, rotate, add drop shadows, place on busy backgrounds without a container
- **On dark backgrounds:** Use the logo as-is (it has a transparent or dark-compatible background)
- **On light backgrounds:** Ensure sufficient contrast; consider a dark container or the inverted version
- **Social avatars:** Use AetherLogo.png cropped to square, centered

### 1.4 Voice and Visual Tone

- Dark, sophisticated aesthetic (think Vercel, Linear, Raycast)
- No gradients unless subtle (brand purple to transparent)
- Clean lines, generous whitespace
- Terminal/code snippets are first-class visual elements
- The ant colony metaphor should feel elegant, not cartoonish
- Avoid: stock photos, generic tech illustrations, clip art

---

## 2. Existing Asset Inventory

### 2.1 Root-Level Assets

| File | Dimensions | Format | Size | Description | Recommended Usage |
|------|-----------|--------|------|-------------|-------------------|
| `AetherLogo.png` | 1922 x 1618 | PNG (RGBA) | ~991 KB | Primary logo mark. Appears to be the Aether logo with ant/colony iconography. Aspect ratio is roughly 1.19:1 (slightly wider than tall). | GitHub repo avatar, favicon, social media profile pictures, navbar logo, watermark |
| `AetherBanner.png` | 3168 x 1344 | PNG (RGBA) | ~6.7 MB | Wide banner image. Aspect ratio 2.36:1 (ultra-wide). Large file size -- will need optimization for web. | GitHub README header (currently used), landing page hero background, social media cover images |
| `AetherAnts.png` | 1200 x 896 | PNG (RGBA) | ~2.8 MB | Illustration of ants. Aspect ratio 1.34:1. | Feature illustrations, social media posts, "About" section imagery, Reddit/discord embed thumbnails |
| `AetherColonyArt.png` | 2816 x 1536 | PNG (RGBA) | ~9.6 MB | Colony artwork/illustration. Aspect ratio 1.83:1 (near 16:9). | Landing page hero, about section, printed materials, presentation slides |

### 2.2 Asset Quality Assessment

**Strengths:**
- All assets are high-resolution PNG with RGBA transparency
- Four distinct visual pieces provide variety
- The banner is already deployed in the README successfully

**Issues to Address:**
- All files are very large for web use (total ~20 MB). Web-optimized versions are critical.
- AetherLogo.png is not square -- needs a square crop for avatar/profile use
- AetherBanner.png at 6.7 MB will slow page load significantly
- No SVG versions exist for scalable/vector use
- No favicon or ICO file exists
- No white/background variants of the logo for light backgrounds

### 2.3 Derivative Assets Needed from Existing Files

| Source | Derivative | Specs | Priority |
|--------|-----------|-------|----------|
| `AetherLogo.png` | Square crop (centered) | 512x512 PNG | MUST-HAVE |
| `AetherLogo.png` | Square crop (centered) | 192x192 PNG | MUST-HAVE |
| `AetherLogo.png` | Favicon | 32x32, 16x16 ICO/PNG | MUST-HAVE |
| `AetherLogo.png` | Light background variant | 512x512 PNG (inverted or on white) | NICE-TO-HAVE |
| `AetherLogo.png` | SVG trace | Scalable vector | HIGH |
| `AetherBanner.png` | Web-optimized banner | Max 1200px wide, < 200 KB, WebP + PNG fallback | MUST-HAVE |
| `AetherBanner.png` | OG Image (1200x630) | 1200x630 PNG/JPG | MUST-HAVE |
| `AetherBanner.png` | Twitter card image | 1200x675 PNG/JPG | MUST-HAVE |
| `AetherAnts.png` | Web-optimized | Max 800px wide, < 150 KB, WebP + PNG fallback | HIGH |
| `AetherColonyArt.png` | Web-optimized hero | Max 1600px wide, < 300 KB, WebP + PNG fallback | HIGH |
| `AetherColonyArt.png` | Thumbnail version | 400x225 PNG | HIGH |

---

## 3. Missing Assets by Channel

### 3.1 GitHub Repository

| Asset | Description | Priority |
|-------|-------------|----------|
| **OG Image** | Open Graph image for social previews when repo link is shared. GitHub uses `og:image` from the repo. 1200x630. Shows logo, tagline, "Open Source Multi-Agent AI Orchestration" | MUST-HAVE |
| **Social Preview image** | GitHub-specific social preview (same as OG image) | MUST-HAVE |
| **Demo GIF** | Animated GIF showing the colony lifecycle: init, plan, build, continue, seal. Terminal recording, 15-30 seconds, max 10 MB. Shows real commands and output. | HIGH |
| **Architecture diagram** | Clean, branded SVG/PNG showing the colony architecture: Queen, castes, pheromone signals, memory pipeline. Suitable for README embedding. | HIGH |
| **Contributor badge** | Small badge/graphic for contributors to display | LOW |
| **GitHub Action badge** | Status badges are text-based (shields.io), but a custom "Powered by Aether" badge for repos using Aether would be useful | LOW |

### 3.2 Social Media

| Asset | Channel | Description | Priority |
|-------|---------|-------------|----------|
| **Twitter/X Card Image** | Twitter/X | 1200x675. Logo + tagline. Used when github.com link is shared. | MUST-HAVE |
| **Twitter/X Profile Assets** | Twitter/X | Profile photo (400x400), header banner (1500x500) | HIGH |
| **LinkedIn Banner** | LinkedIn | 1584x396 for personal profile OR 1128x191 for company page. Logo + tagline + URL | HIGH |
| **LinkedIn Post Image** | LinkedIn | 1200x627. For link previews when sharing GitHub URL. | MUST-HAVE |
| **Reddit Post Thumbnail** | Reddit | Square (ideally). Used when sharing links. Should work at multiple sizes. AetherAnts.png could serve this role if optimized. | HIGH |
| **Discord Server Icon** | Discord | 512x512 PNG. Server avatar. | HIGH |
| **Discord Invite Banner** | Discord | 1920x1080. Invite splash image. | NICE-TO-HAVE |
| **Generic social share image** | All | 1200x630. Universal OG image that works across all platforms | MUST-HAVE |

### 3.3 Landing Page (aetherantcolony.com)

| Asset | Section | Description | Priority |
|-------|---------|-------------|----------|
| **Hero background** | Hero | Animated colony or subtle particle background. Could use AetherColonyArt.png with CSS overlay. 1920x1080 min. | HIGH |
| **Hero logo** | Hero | Clean logo placement, possibly with subtle glow animation | MUST-HAVE |
| **Feature card icons** | Features grid | 6 custom icons (64x64 SVG) for: Colony Architecture, Pheromone Signals, Structural Learning, Autopilot Mode, Skills System, Context Continuity. Described in landing-hero.md. | HIGH |
| **"How It Works" step icons** | How It Works | 3 icons (64x64 SVG) for: Init, Plan, Build steps | HIGH |
| **CTA section background** | CTA | Dark gradient or subtle pattern. Could be CSS-only. | LOW |
| **Footer logo** | Footer | Small logo mark, white or brand color on dark background | MUST-HAVE |
| **Favicon** | Browser | 32x32 + 16x16 + 180x180 (Apple Touch) | MUST-HAVE |
| **Testimonial avatars** | Social proof | Placeholder avatar circles until real testimonials exist | LOW |

### 3.4 Product Hunt

| Asset | Description | Priority |
|-------|-------------|----------|
| **Product Hunt thumbnail** | 240x240. First impression. Logo + tagline. Clean, readable at small size. | MUST-HAVE |
| **Gallery image 1** | "Launch a colony in one command" -- Screenshot or mockup of the init screen | HIGH |
| **Gallery image 2** | "24 workers, zero micromanagement" -- Phase planning view / architecture diagram | HIGH |
| **Gallery image 3** | "Pheromone signals guide the colony" -- Visual of signal system (FOCUS/REDIRECT/FEEDBACK) | HIGH |
| **Gallery image 4** | "Memory that compounds" -- Visual of instinct-to-wisdom-to-Hive-Brain pipeline | NICE-TO-HAVE |
| **Gallery image 5** | "Autopilot: ship while you sleep" -- Terminal output showing unattended build progress | HIGH |

### 3.5 Documentation and Content

| Asset | Description | Priority |
|-------|-------------|----------|
| **Animated demo** | Full 60-120 second walkthrough video. From blank directory to working API. Terminal recording with asciinema or terminalizer. Hosted on YouTube. Linked from README. | HIGH |
| **Quick-start GIF** | Short 10-second GIF showing the five core commands. Embeddable in README. | HIGH |
| **Comparison infographic** | Visual comparison of Aether vs CrewAI vs AutoGen vs LangGraph. Could be an enhanced version of the README comparison table. | NICE-TO-HAVE |
| **Pheromone signal diagram** | Clean diagram showing how FOCUS, REDIRECT, FEEDBACK signals flow from user to workers. | HIGH |
| **Memory pipeline diagram** | Visual showing Observation -> Instinct -> Wisdom -> Hive Brain flow. | HIGH |

---

## 4. Detailed Asset Specifications

### 4.1 MUST-HAVE Assets (Ship Blockers)

#### ASSET: Universal OG Image
- **Filename:** `og-image.png`
- **Dimensions:** 1200 x 630 px
- **Format:** PNG (or JPG if file size is an issue)
- **Max file size:** 300 KB
- **Content:**
  - Aether logo (centered, prominent)
  - Tagline: "The whole is greater than the sum of its ants."
  - Subtitle: "Open-source multi-agent AI orchestration"
  - URL: "aetherantcolony.com"
  - Dark background matching brand (`#0a0a14` or `#1a1a2e`)
- **Usage:** GitHub social preview, LinkedIn link preview, Discord link embed, any platform that fetches OG meta tags
- **Production notes:** Derive from AetherBanner.png (crop and resize). Or create new composite. Test with GitHub's social preview debugger and opengraph.xyz.

#### ASSET: Favicon Set
- **Filenames:** `favicon-16x16.png`, `favicon-32x32.png`, `apple-touch-icon.png`, `favicon.ico`
- **Dimensions:** 16x16, 32x32, 180x180, multi-size ICO
- **Format:** PNG + ICO
- **Content:** AetherLogo.png cropped to square, scaled down. Ensure readability at 16px.
- **Usage:** Browser tabs, bookmarks, Apple Touch for iOS home screen
- **Production notes:** The ant/colony icon element of AetherLogo.png should be isolated for small sizes. At 16px, fine detail will be lost -- simplify if needed.

#### ASSET: Product Hunt Thumbnail
- **Filename:** `producthunt-thumbnail.png`
- **Dimensions:** 240 x 240 px
- **Format:** PNG
- **Max file size:** 100 KB
- **Content:** Aether logo centered on dark brand background. Must be instantly recognizable at small size.
- **Usage:** Product Hunt listing, first visual impression
- **Production notes:** Test at actual 240px size. Logo must be legible. Consider adding a subtle purple glow or border.

#### ASSET: Web-Optimized Logo Set
- **Filenames:** `logo-dark.png` (on dark bg), `logo-light.png` (on light bg), `logo-square.png` (512x512 avatar)
- **Dimensions:** Variable (logo at ~200px height; square at 512x512)
- **Format:** PNG, ideally also SVG
- **Max file size:** 50 KB each (PNG), 10 KB (SVG)
- **Content:** Clean logo exports from AetherLogo.png
- **Usage:** Navbar, footer, social profiles, embedded in docs
- **Production notes:** Current AetherLogo.png is 991 KB at 1922x1618. Web versions need aggressive optimization. Consider creating an SVG trace for infinite scalability.

#### ASSET: Web-Optimized Banner
- **Filename:** `banner-web.webp` (+ `banner-web.png` fallback)
- **Dimensions:** 1200 x 508 px (maintain 2.36:1 ratio of original, capped at 1200px wide)
- **Format:** WebP primary, PNG fallback
- **Max file size:** 200 KB (WebP), 300 KB (PNG fallback)
- **Content:** Scaled-down version of AetherBanner.png
- **Usage:** README embed, landing page sections
- **Production notes:** Original is 6.7 MB. Use lossy WebP compression or careful PNG optimization (tinypng, squoosh). Consider reducing color depth if quality is maintained.

### 4.2 HIGH Priority Assets

#### ASSET: Twitter/X Profile and Header
- **Filenames:** `twitter-avatar.png`, `twitter-header.png`
- **Dimensions:** 400x400 (avatar), 1500x500 (header)
- **Format:** PNG
- **Content:**
  - Avatar: Square crop of AetherLogo.png centered on dark bg
  - Header: AetherBanner.png or AetherColonyArt.png cropped/extended to 1500x500 with dark overlay and tagline
- **Usage:** @AetherColony (or applicable handle) Twitter/X profile
- **Production notes:** Header should have text/CTA area on the right side (profile pic overlaps the left). Place tagline and URL in the right 2/3.

#### ASSET: LinkedIn Banner
- **Filename:** `linkedin-banner.png`
- **Dimensions:** 1584 x 396 px (personal profile) OR 1128 x 191 px (company page)
- **Format:** PNG
- **Content:** Logo left, tagline center-right, URL right. Clean dark background. Similar style to Twitter header but wider and shorter aspect ratio.
- **Usage:** LinkedIn personal profile banner and/or company page banner
- **Production notes:** Personal profile banner is recommended (more visibility). The safe zone for text is the center ~60% (left and right edges are cropped on mobile).

#### ASSET: Discord Server Icon
- **Filename:** `discord-icon.png`
- **Dimensions:** 512 x 512 px
- **Format:** PNG
- **Content:** AetherLogo.png square crop, optimized for circular mask (Discord crops to circle). Ensure the central icon element is centered and clear.
- **Usage:** Discord server profile picture
- **Production notes:** Discord applies a circular crop. Test with a circular mask to ensure nothing important gets clipped.

#### ASSET: Feature Card Icons (Landing Page)
- **Filenames:** `icon-colony.svg`, `icon-pheromone.svg`, `icon-learning.svg`, `icon-autopilot.svg`, `icon-skills.svg`, `icon-context.svg`
- **Dimensions:** 64 x 64 px (display size), SVG source
- **Format:** SVG (scalable), with PNG fallback at 128x128
- **Content:**
  - Colony Architecture: Network graph / ant colony cross-section / interconnected nodes
  - Pheromone Signals: Concentric ripples / signal waves / broadcast icon
  - Structural Learning: Ascending steps / upward arrow / brain with growth
  - Autopilot Mode: Play button / circular arrow / cruise control metaphor
  - Skills System: Toolbox / interconnected skill nodes / modular blocks
  - Context Continuity: Chain links / unbroken line / bridge
- **Style:** Line icon style (1.5-2px stroke), brand purple (#7B3FE4) on transparent background. Consistent visual weight across all six.
- **Usage:** Landing page features grid (described in landing-hero.md)

#### ASSET: Demo GIF (README)
- **Filename:** `demo.gif` (or `demo.webm` for modern browsers)
- **Dimensions:** 800 x 400 px (terminal recording)
- **Format:** GIF (max 10 MB) or WebM (preferred for quality)
- **Duration:** 15-30 seconds
- **Content:** Terminal recording showing the five-command lifecycle:
  1. `/ant-init "Build a REST API for task management"` -- colony initializes
  2. `/ant-plan` -- phases generated
  3. `/ant-build 1` -- workers deployed, output scrolling
  4. `/ant-continue` -- verification and advancement
  5. Final state showing passing tests / completed phase
- **Style:** Dark terminal theme (matching GitHub dark). Font: JetBrains Mono or similar. Clean, no typos, no excessive scrolling.
- **Usage:** README embed below Quick Start section
- **Production tools:** asciinema (record) + asciicast2gif or svg-term (convert). Or terminalizer. Or record screen and convert with ffmpeg.

#### ASSET: Animated Demo Video
- **Filename:** `demo.mp4` (hosted on YouTube)
- **Dimensions:** 1920 x 1080 px (or 1280x720 for smaller file)
- **Format:** MP4 (H.264)
- **Duration:** 60-120 seconds
- **Content:** Full walkthrough from blank directory to working API:
  1. Install (`go install`)
  2. Create project directory
  3. `/ant-lay-eggs`
  4. `/ant-init` with goal
  5. `/ant-colonize` (if existing code)
  6. `/ant-plan` -- show phase output
  7. `/ant-focus` and `/ant-redirect` signals
  8. `/ant-build 1` -- show parallel workers
  9. `/ant-continue` -- verification
  10. `/ant-run` (autopilot demo)
  11. Final result: working project
- **Audio:** Optional narration or text overlays. No music (keeps it professional).
- **Usage:** YouTube embed in README, linked from landing page, shared on social media
- **Production tools:** asciinema + agg, OBS screen record, or dedicated terminal recorder

#### ASSET: Pheromone Signal Diagram
- **Filename:** `diagram-pheromones.svg` (+ PNG export)
- **Dimensions:** 800 x 500 px (flexible, SVG)
- **Format:** SVG primary, PNG fallback
- **Content:** Visual flow showing:
  - User emits signal (left): FOCUS / REDIRECT / FEEDBACK
  - Signal propagates through colony (center): visualized as waves/trails
  - Workers receive and adapt (right): builders, watchers, scouts with signal indicators
  - Signal decay (bottom): fading trail showing TTL/expiration
- **Style:** Clean diagram, brand purple for signals, dark background, consistent with README aesthetic
- **Usage:** README "Pheromone Signals" section, landing page, presentation slides

#### ASSET: Memory Pipeline Diagram
- **Filename:** `diagram-memory.svg` (+ PNG export)
- **Dimensions:** 800 x 400 px (flexible, SVG)
- **Format:** SVG primary, PNG fallback
- **Content:** Flow diagram showing:
  - Raw Observations -> Trust-Scored Instincts (0.2-1.0) -> QUEEN.md Wisdom (0.80+) -> Hive Brain (cross-project)
  - Visual showing confidence score progression
  - Arrows showing promotion/demotion paths
- **Style:** Same visual language as pheromone diagram
- **Usage:** README "Structural Learning" section, landing page

#### ASSET: Architecture Diagram
- **Filename:** `diagram-architecture.svg` (+ PNG export)
- **Dimensions:** 1000 x 700 px (flexible, SVG)
- **Format:** SVG primary, PNG fallback
- **Content:** Colony architecture showing:
  - Queen (central coordination)
  - Caste groups: Builders, Probes, Watchers, Scouts, etc.
  - Pheromone signal flow between castes
  - Memory system (instincts, wisdom, hive brain)
  - External interfaces (Claude Code, OpenCode)
- **Style:** Hierarchical or network layout. Brand colors. Professional technical diagram.
- **Usage:** README, landing page, documentation, presentations

### 4.3 NICE-TO-HAVE Assets

#### ASSET: Discord Invite Banner
- **Filename:** `discord-invite.png`
- **Dimensions:** 1920 x 1080 px
- **Format:** PNG
- **Content:** AetherColonyArt.png or custom composite with "Join the Colony" CTA and Discord URL
- **Usage:** Discord server invite splash page

#### ASSET: Comparison Infographic
- **Filename:** `infographic-comparison.png`
- **Dimensions:** 1200 x 800 px
- **Format:** PNG
- **Content:** Visual comparison table/chart: Aether vs CrewAI vs AutoGen vs LangGraph. Feature comparison with checkmarks, scores, or visual indicators.
- **Usage:** Blog posts, social media, README enhancement

#### ASSET: Animated Landing Page Background
- **Type:** CSS animation or lightweight canvas/SVG animation
- **Content:** Subtle particle system or ant trail animation. Should not distract from content or impact performance. Dark background with small glowing particles moving in trail patterns.
- **Usage:** Landing page hero section background
- **Production notes:** Keep under 50 KB total (JS + CSS). Use requestAnimationFrame. Respect `prefers-reduced-motion`.

#### ASSET: "Powered by Aether" Badge
- **Filename:** `powered-by-aether.svg`
- **Dimensions:** 200 x 40 px (flexible, SVG)
- **Format:** SVG
- **Content:** Small ant icon + "Powered by Aether" text
- **Usage:** Repos/projects built with Aether can display this badge

#### ASSET: Sticker Pack
- **Format:** PNG or SVG
- **Content:** Aether-themed stickers: individual caste icons, pheromone signal symbols, the ant logo in various poses
- **Usage:** Community swag, GitHub Sponsors rewards, conference handouts

#### ASSET: Light Background Logo Variant
- **Filename:** `logo-light-bg.png`
- **Dimensions:** 512 x 512 px
- **Format:** PNG
- **Content:** AetherLogo.png adjusted for use on white/light backgrounds. May require outline, shadow, or color inversion.
- **Usage:** Print materials, light-themed websites, partner logos

---

## 5. Priority Matrix

### Tier 1: Ship Blockers (Must have before public launch)

| # | Asset | Effort | Impact |
|---|-------|--------|--------|
| 1 | Universal OG Image (1200x630) | Low (derive from existing) | Very High |
| 2 | Favicon set (16+32+180+ICO) | Low (crop/scale existing) | High |
| 3 | Web-optimized banner (< 200 KB) | Low (compress existing) | High |
| 4 | Product Hunt thumbnail (240x240) | Low (crop existing) | High |
| 5 | Web-optimized logo square (512x512) | Low (crop/scale existing) | High |
| 6 | Discord server icon (512x512) | Low (crop existing) | Medium |

**Estimated effort:** 1-2 hours (all derivable from existing assets with basic image editing)

### Tier 2: High Value (First week post-launch)

| # | Asset | Effort | Impact |
|---|-------|--------|--------|
| 7 | Twitter/X profile + header | Medium | High |
| 8 | LinkedIn banner | Medium | High |
| 9 | Demo GIF (README) | Medium | Very High |
| 10 | Pheromone signal diagram (SVG) | Medium-High | High |
| 11 | Memory pipeline diagram (SVG) | Medium | High |
| 12 | Architecture diagram (SVG) | High | Very High |
| 13 | Feature card icons (6x SVG) | Medium | Medium |
| 14 | How It Works step icons (3x SVG) | Low-Medium | Medium |
| 15 | Web-optimized AetherAnts.png | Low (compress existing) | Medium |
| 16 | Web-optimized AetherColonyArt.png | Low (compress existing) | Medium |
| 17 | Animated demo video (60-120s) | High | Very High |

**Estimated effort:** 8-16 hours (mix of derivation and new creation)

### Tier 3: Polish (Second week+)

| # | Asset | Effort | Impact |
|---|-------|--------|--------|
| 18 | SVG logo trace | Medium | High (long-term) |
| 19 | Discord invite banner | Low-Medium | Low |
| 20 | Product Hunt gallery images (5x) | Medium | Medium |
| 21 | Comparison infographic | Medium | Medium |
| 22 | Light background logo variant | Low | Low |
| 23 | Animated landing page background | Medium-High | Medium |
| 24 | "Powered by Aether" badge (SVG) | Low | Low |
| 25 | Sticker pack | Medium | Low (community) |
| 26 | Reddit thumbnail (dedicated) | Low | Medium |

**Estimated effort:** 10-20 hours

---

## 6. Production Notes

### 6.1 Recommended Tools

| Task | Tool | Notes |
|------|------|-------|
| Image optimization | Squoosh (CLI/GUI), TinyPNG, ImageOptim | Target 70-80% size reduction |
| SVG creation | Figma, Inkscape, Illustrator | Figma is free for individuals |
| Terminal recording | asciinema + svg-term or agg | Produces clean, crisp terminal output |
| Screen recording | OBS, CleanShot X (macOS) | For demo video |
| Video editing | DaVinci Resolve (free), FFmpeg CLI | For demo video post-production |
| GIF conversion | gifski, ffmpeg | gifski produces smallest high-quality GIFs |
| Favicon generation | realfavicongenerator.net | Generates all sizes + ICO + manifest |
| OG image testing | opengraph.xyz, cards-dev.twitter.com | Validate before deploying |

### 6.2 File Organization

Recommended directory structure for assets:

```
/Users/callumcowie/repos/Aether/
  assets/
    logo/
      aether-logo.svg              # SVG master
      aether-logo-dark.png         # For dark backgrounds (200px height)
      aether-logo-light.png        # For light backgrounds (200px height)
      aether-logo-square.png       # 512x512 square crop
      aether-logo-square-sm.png    # 192x192 square crop
      favicon-16x16.png
      favicon-32x32.png
      favicon.ico
      apple-touch-icon.png
    banner/
      aether-banner.webp           # Web-optimized banner
      aether-banner.png            # PNG fallback
    og/
      og-image.png                 # Universal social preview (1200x630)
    social/
      twitter-avatar.png           # 400x400
      twitter-header.png           # 1500x500
      linkedin-banner.png          # 1584x396
      discord-icon.png             # 512x512
      discord-invite.png           # 1920x1080
      reddit-thumbnail.png         # Square
      producthunt-thumbnail.png    # 240x240
    illustrations/
      aether-ants.webp             # Web-optimized
      aether-ants.png              # PNG fallback
      aether-colony-art.webp       # Web-optimized
      aether-colony-art.png        # PNG fallback
    diagrams/
      architecture.svg             # Colony architecture
      architecture.png             # PNG export
      pheromones.svg               # Signal system
      pheromones.png               # PNG export
      memory-pipeline.svg          # Learning system
      memory-pipeline.png          # PNG export
    icons/
      icon-colony.svg
      icon-pheromone.svg
      icon-learning.svg
      icon-autopilot.svg
      icon-skills.svg
      icon-context.svg
      icon-init.svg
      icon-plan.svg
      icon-build.svg
      powered-by-aether.svg
    demo/
      demo.gif                     # Short README demo
      demo.webm                    # Modern browser alternative
    video/
      demo.mp4                     # Full walkthrough (hosted on YouTube)
```

### 6.3 Web Performance Targets

| Asset Type | Max Size | Format Priority |
|------------|----------|-----------------|
| Logos/icons | 50 KB | SVG > PNG > WebP |
| Banners/hero images | 200 KB | WebP > AVIF > JPEG |
| Diagrams | 100 KB | SVG (inline or linked) |
| GIFs | 5 MB (2 MB preferred) | Consider WebM instead |
| Favicon | 10 KB | ICO + PNG |

### 6.4 Accessibility Considerations

- All images must have descriptive `alt` text
- SVGs should include `<title>` and `<desc>` elements
- Diagrams should be understandable without color alone (use patterns/labels)
- Animated assets should respect `prefers-reduced-motion`
- Terminal demos should use a font size readable at default zoom (14-16px)
- Color contrast ratios: text on backgrounds must meet WCAG AA (4.5:1 for normal text, 3:1 for large text)

### 6.5 Version Control

- Source files (SVG, Figma links) should be committed to the repo
- Generated/optimized files (WebP, ICO) should also be committed for easy deployment
- Consider adding an `assets/README.md` with usage guidelines for contributors
- Large video files should NOT be committed; host on YouTube and link

---

## Appendix: Quick Reference Checklist

### Before Launch
- [ ] Universal OG image created and tested (opengraph.xyz)
- [ ] Favicon set generated and deployed
- [ ] Web-optimized banner replacing 6.7 MB original in README
- [ ] Product Hunt thumbnail ready
- [ ] Square logo avatar ready for social profiles
- [ ] Discord server icon uploaded

### Week 1 Post-Launch
- [ ] Twitter/X profile and header configured
- [ ] LinkedIn banner uploaded
- [ ] Demo GIF embedded in README
- [ ] At least one architecture/feature diagram created
- [ ] Demo video recorded and uploaded to YouTube
- [ ] All root PNGs have web-optimized versions

### Week 2+ Post-Launch
- [ ] SVG logo created for scalability
- [ ] Full diagram set (architecture, pheromones, memory)
- [ ] Feature card icons for landing page
- [ ] Product Hunt gallery images
- [ ] Animated landing page background (if applicable)
