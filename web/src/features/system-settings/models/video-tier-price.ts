/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
// Video tier pricing shared between the admin editor and its tests.
// Stored as an option JSON: { [model]: { [tierKey]: price } } where tierKey is
// "<resolution>" (without video input) or "<resolution>_video" (with video input).
// Prices are entered in CNY per 1M tokens (the Ark price-list unit); billing only
// uses their ratios against the base tier.

export const VIDEO_TIER_RESOLUTIONS = [
  '480p',
  '720p',
  '768p',
  '1080p',
  '2k',
  '4k',
] as const

export type VideoTierResolution = (typeof VIDEO_TIER_RESOLUTIONS)[number]
export type VideoTierKey =
  | VideoTierResolution
  | `${VideoTierResolution}_video`

// Editor state keeps raw input strings so partially typed values (e.g. "1.")
// are not clobbered; numeric conversion happens when serializing or saving.
export type VideoTierPriceEntry = {
  model: string
  prices: Partial<Record<VideoTierKey, string>>
}

export type VideoTierPriceParseResult =
  | { ok: true; entries: VideoTierPriceEntry[] }
  | { ok: false; error: string }

export type VideoTierPriceBuildResult =
  | { ok: true; json: string }
  | { ok: false; error: string }

const VIDEO_TIER_KEY_PATTERN = /^(480p|720p|768p|1080p|2k|4k)(_video)?$/

export function videoTierKey(
  resolution: VideoTierResolution,
  hasVideo: boolean
): VideoTierKey {
  return hasVideo ? `${resolution}_video` : resolution
}

function isVideoTierKey(key: string): key is VideoTierKey {
  return VIDEO_TIER_KEY_PATTERN.test(key)
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export function parseVideoTierPriceJson(
  json: string
): VideoTierPriceParseResult {
  const trimmed = json.trim()
  if (!trimmed) {
    return { ok: true, entries: [] }
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    return { ok: false, error: 'Invalid JSON' }
  }

  if (!isPlainObject(parsed)) {
    return { ok: false, error: 'Expected a JSON object of model → tier prices' }
  }

  const entries: VideoTierPriceEntry[] = []
  for (const [model, rawPrices] of Object.entries(parsed)) {
    const name = model.trim()
    if (!name) {
      return { ok: false, error: 'Model name cannot be empty' }
    }
    if (!isPlainObject(rawPrices)) {
      return {
        ok: false,
        error: 'Each model entry must be an object of tier prices',
      }
    }

    const prices: Partial<Record<VideoTierKey, string>> = {}
    for (const [tier, rawValue] of Object.entries(rawPrices)) {
      if (!isVideoTierKey(tier)) {
        return { ok: false, error: 'Contains an unknown tier key' }
      }
      if (
        typeof rawValue !== 'number' ||
        !Number.isFinite(rawValue) ||
        rawValue <= 0
      ) {
        return { ok: false, error: 'Prices must be positive numbers' }
      }
      prices[tier] = String(rawValue)
    }
    entries.push({ model: name, prices })
  }

  return { ok: true, entries }
}

export function buildVideoTierPriceJson(
  entries: VideoTierPriceEntry[]
): VideoTierPriceBuildResult {
  if (entries.length === 0) {
    return { ok: true, json: '{}' }
  }

  const seen = new Set<string>()
  const map: Record<string, Record<string, number>> = {}

  for (const entry of entries) {
    const name = entry.model.trim()
    if (!name) {
      return { ok: false, error: 'Model name cannot be empty' }
    }
    const normalizedName = name.toLowerCase()
    if (seen.has(normalizedName)) {
      return { ok: false, error: 'Duplicate model name' }
    }
    seen.add(normalizedName)

    const prices: Record<string, number> = {}
    for (const resolution of VIDEO_TIER_RESOLUTIONS) {
      for (const hasVideo of [false, true]) {
        const key = videoTierKey(resolution, hasVideo)
        const raw = entry.prices[key]?.trim() ?? ''
        if (!raw) continue
        const value = Number(raw)
        if (!Number.isFinite(value) || value <= 0) {
          return { ok: false, error: 'Prices must be positive numbers' }
        }
        prices[key] = value
      }
    }
    map[name] = prices
  }

  return { ok: true, json: JSON.stringify(map, null, 2) }
}
