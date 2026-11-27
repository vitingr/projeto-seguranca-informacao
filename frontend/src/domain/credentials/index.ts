export const LOGIN_PATTERN = /^[a-z0-9._-]{3,64}$/

export const passwordLength = (value: string) =>
  [...value.normalize('NFKC')].length
