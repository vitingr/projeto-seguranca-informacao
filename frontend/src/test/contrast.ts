import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const GLOBALS = join(process.cwd(), 'src', 'styles', 'globals.css')

export const themeToken = (name: string): string => {
  const css = readFileSync(GLOBALS, 'utf8')
  const match = new RegExp(`--${name}:\\s*(oklch\\([^)]*\\))`).exec(css)
  if (!match) throw new Error(`token --${name} not found in globals.css`)
  return match[1]
}

const linearSrgb = (oklch: string): [number, number, number] => {
  const parts = /oklch\(\s*([\d.]+)\s+([\d.]+)\s+([\d.]+)\s*\)/.exec(oklch)
  if (!parts) throw new Error(`not an oklch() color: ${oklch}`)
  const [lightness, chroma, hue] = parts.slice(1).map(Number)
  const a = chroma * Math.cos((hue * Math.PI) / 180)
  const b = chroma * Math.sin((hue * Math.PI) / 180)
  const l = (lightness + 0.3963377774 * a + 0.2158037573 * b) ** 3
  const m = (lightness - 0.1055613458 * a - 0.0638541728 * b) ** 3
  const s = (lightness - 0.0894841775 * a - 1.291485548 * b) ** 3
  const clamp = (value: number) => Math.min(1, Math.max(0, value))
  return [
    clamp(4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s),
    clamp(-1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s),
    clamp(-0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s)
  ]
}

const luminance = (oklch: string): number => {
  const [red, green, blue] = linearSrgb(oklch)
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue
}

export const contrastRatio = (first: string, second: string): number => {
  const [light, dark] = [luminance(first), luminance(second)].sort(
    (x, y) => y - x
  )
  return (light + 0.05) / (dark + 0.05)
}
