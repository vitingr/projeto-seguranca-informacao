// One fixed time zone, so the server render and the browser hydration print
// the same text whatever the machine's zone is.
const FORMAT = new Intl.DateTimeFormat('pt-BR', {
  dateStyle: 'short',
  timeStyle: 'short',
  timeZone: 'America/Sao_Paulo'
})

/** An ISO 8601 timestamp as "dd/mm/aaaa, hh:mm" (Brasília time). */
export const formatDateTime = (iso: string): string => {
  const date = new Date(iso)
  return Number.isNaN(date.getTime()) ? '' : FORMAT.format(date)
}
