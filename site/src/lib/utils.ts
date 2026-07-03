/**
 * Copyright (c) 2026 Ali Sait Teke
 * SPDX-License-Identifier: MIT
 */

/**
 * Tailwind class merge helper used by shadcn/ui components.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-1.0-scaffold.md
 */
import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
