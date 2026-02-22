---
last_reviewed: YYYY-MM-DD
---

# PRODUCT.md Template

> Copy this file to your project root as `PRODUCT.md` and fill in each section.
> When present, `/pre-mortem` and `/vibe` automatically include product perspectives in council reviews.

## Mission

<!-- One sentence: what does this product do and for whom? -->

## Target Personas

<!-- 2-3 user personas. For each: role, goal, key pain point. -->

### Persona 1: [Role]
- **Goal:** What they're trying to accomplish
- **Pain point:** What makes this hard today

### Persona 2: [Role]
- **Goal:** What they're trying to accomplish
- **Pain point:** What makes this hard today

## Core Value Propositions

<!-- What makes this product worth using? 2-4 bullet points. -->

-

## Competitive Landscape

<!-- How does this differ from alternatives? What's the unique angle? -->

| Alternative | Strength | Our Differentiation |
|-------------|----------|---------------------|
| | | |

## Usage

This file enables product-aware council reviews:

- **`/pre-mortem`** — Automatically includes `product` perspectives (user-value, adoption-barriers, competitive-position) alongside plan-review judges when this file exists.
- **`/vibe`** — Automatically includes `developer-experience` perspectives (api-clarity, error-experience, discoverability) alongside code-review judges when this file exists.
- **`/council --preset=product`** — Run product review on demand.
- **`/council --preset=developer-experience`** — Run DX review on demand.

Explicit `--preset` overrides from the user skip auto-include (user intent takes precedence).
