// Dates shown to the user: Brazilian format, in the São Paulo time zone,
// whatever the time zone of the browser (the server sends UTC).

const TIME_ZONE = 'America/Sao_Paulo'

const DATE = new Intl.DateTimeFormat('pt-BR', {
  day: '2-digit',
  month: '2-digit',
  timeZone: TIME_ZONE,
  year: 'numeric'
})

const TIME = new Intl.DateTimeFormat('pt-BR', {
  hour: '2-digit',
  hourCycle: 'h23',
  minute: '2-digit',
  timeZone: TIME_ZONE
})

/** "25/11/2026", or `null` when `iso` is not a date. */
export const formatDate = (iso: string): null | string => {
  const date = new Date(iso)
  return Number.isNaN(date.getTime()) ? null : DATE.format(date)
}

/** "25/11/2026 às 11:30", or `null` when `iso` is not a date. */
export const formatDateTime = (iso: string): null | string => {
  const date = new Date(iso)
  return Number.isNaN(date.getTime())
    ? null
    : `${DATE.format(date)} às ${TIME.format(date)}`
}

const toDate = (value: Date | string): Date | null => {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

/** "14:12" in São Paulo, for "Atualizado às" (spec 003 D-4); `null` if not a date. */
export const formatClock = (iso: string): null | string => {
  const date = toDate(iso)
  return date ? TIME.format(date) : null
}

/**
 * The time of a thread entry (spec 003 section 7, "The time"): "14:04" on
 * the same São Paulo day as `now`, "25/11 14:04" earlier in the same year,
 * "25/11/2025 14:04" otherwise; `null` when either value is not a date.
 */
export const formatThreadTime = (
  iso: string,
  now: Date | string
): null | string => {
  const date = toDate(iso)
  const today = toDate(now)
  if (!date || !today) return null
  const [day, month, year] = DATE.format(date).split('/')
  const [todayDay, todayMonth, todayYear] = DATE.format(today).split('/')
  const time = TIME.format(date)
  if (year !== todayYear) return `${day}/${month}/${year} ${time}`
  if (day === todayDay && month === todayMonth) return time
  return `${day}/${month} ${time}`
}
