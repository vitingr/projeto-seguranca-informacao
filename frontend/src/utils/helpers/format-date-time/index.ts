// One fixed time zone, so the server render and the browser hydration print
// the same text whatever the machine's zone is.
const TIME_ZONE = 'America/Sao_Paulo'

const FORMAT = new Intl.DateTimeFormat('pt-BR', {
  dateStyle: 'short',
  timeStyle: 'short',
  timeZone: TIME_ZONE
})

const TIME = new Intl.DateTimeFormat('pt-BR', {
  timeStyle: 'short',
  timeZone: TIME_ZONE
})

const DAY = new Intl.DateTimeFormat('pt-BR', {
  day: 'numeric',
  month: 'long',
  timeZone: TIME_ZONE,
  weekday: 'long',
  year: 'numeric'
})

const KEY = new Intl.DateTimeFormat('en-CA', {
  day: '2-digit',
  month: '2-digit',
  timeZone: TIME_ZONE,
  year: 'numeric'
})

const parse = (iso: string): Date | null => {
  const date = new Date(iso)
  return Number.isNaN(date.getTime()) ? null : date
}

/** An ISO 8601 timestamp as "dd/mm/aaaa, hh:mm" (Brasília time). */
export const formatDateTime = (iso: string): string => {
  const date = parse(iso)
  return date ? FORMAT.format(date) : ''
}

/** Only the time of day, "hh:mm" (Brasília time). */
export const formatTime = (iso: string): string => {
  const date = parse(iso)
  return date ? TIME.format(date) : ''
}

/** The day, spelled out: "domingo, 22 de novembro de 2026" (Brasília time). */
export const formatDay = (iso: string): string => {
  const date = parse(iso)
  return date ? DAY.format(date) : ''
}

/** "aaaa-mm-dd" of the Brasília day, to group timestamps by day. */
export const dayKey = (iso: string): string => {
  const date = parse(iso)
  return date ? KEY.format(date) : ''
}
