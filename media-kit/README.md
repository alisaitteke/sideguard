# SideGuard Media Kit

Press and launch assets for **Product Hunt**, **Hacker News**, **Twitter/X**, **LinkedIn**, directories, and blog features.

Regenerate PNGs after brand changes:

```bash
cd site && npm run render:media-kit
```

Full asset pipeline (favicons, social cards, media kit):

```bash
cd site && npm run render:assets
```

---

## Quick links

| Resource | URL |
| -------- | --- |
| Website | https://sideguard.io |
| GitHub | https://github.com/alisaitteke/sideguard |
| Install script | https://sideguard.io/setup.sh |
| Maker | [Ali Sait Teke](https://alisait.com) |

Ready-made copy: [`copy.md`](./copy.md)  
Brand tokens (JSON): [`brand-colors.json`](./brand-colors.json)

---

## Folder map

| Folder | Contents |
| ------ | -------- |
| `logos/` | SVG marks + square PNGs (240–1024 px) |
| `icons/` | App-style square icons (180, 512 px) |
| `banners/` | Social / OG / Twitter / LinkedIn headers |
| `gallery/` | Product Hunt gallery slides (1270×760) |

---

## Platform cheat sheet

### Product Hunt

| Field | Asset / value |
| ----- | ------------- |
| **Thumbnail** (240×240) | `logos/thumbnail-240.png` |
| **Gallery** (1270×760, min 2) | `gallery/01-hero-1270x760.png` … `04-install-1270x760.png` |
| **Tagline** (≤60 chars) | See `copy.md` → Product Hunt tagline |
| **Description** | See `copy.md` → Product Hunt description |
| **Website** | https://sideguard.io |
| **Pricing** | Free / open source |

Upload order suggestion: `01-hero` → `02-approval` → `03-integrations` → `04-install`.

### Twitter / X

| Use | Asset |
| --- | ----- |
| Announcement image (16:9) | `banners/twitter-1600x900.png` |
| Open Graph fallback | `banners/og-1200x630.png` |

### LinkedIn

| Use | Asset |
| --- | ----- |
| Post image | `banners/linkedin-1200x627.png` |

### GitHub / README

| Use | Asset |
| --- | ----- |
| Repo social preview | `.github/social-preview.png` (via `render:social-card`) |
| README banner | `assets/readme-hero.png` (via `render:social-card`) |

### General press

| Use | Asset |
| --- | ----- |
| Logo on dark background | `logos/logo-on-dark-512.png` or `logo-on-dark-1024.png` |
| Logo transparent (teal) | `logos/logo-teal-light-512.png` (light UI) |
| Logo transparent (mint) | `logos/logo-teal-dark-512.png` (dark UI) |
| Vector logo (editable) | `logos/logo-teal-dark.svg` or `logo-teal-light.svg` |

---

## Brand colors

| Token | Hex | Usage |
| ----- | --- | ----- |
| Logo (light mode) | `#0d9488` | Light backgrounds, favicon |
| Logo (dark mode) | `#5eead4` | Dark hero, gallery slides |
| Hero background | `#10161c` | Banners, thumbnails |
| Primary green | `#3a9e6e` | Eyebrow, accents |
| Foreground | `#e6edf3` | Headlines on dark |
| Muted text | `#94a3b8` | Body copy on dark |

**Font:** [Geist Variable](https://fontsource.org/fonts/geist) (same as sideguard.io)

**Logo usage**

- Prefer shield + checkmark mark with teal/mint fill on dark backgrounds.
- Do not stretch, rotate, or add effects beyond the subtle glow used in kit assets.
- Minimum clear space: height of the checkmark circle on all sides.

---

## Contact

**Ali Sait Teke** — [alisait.com](https://alisait.com) · [GitHub](https://github.com/alisaitteke) · [LinkedIn](https://www.linkedin.com/in/alisait/)

For press inquiries, use the links above or open a GitHub issue.
