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

export const formatDate = (iso: string): null | string => {
  const date = new Date(iso)
  return Number.isNaN(date.getTime()) ? null : DATE.format(date)
}

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

export const formatClock = (iso: string): null | string => {
  const date = toDate(iso)
  return date ? TIME.format(date) : null
}

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
