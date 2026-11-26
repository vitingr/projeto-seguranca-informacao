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
