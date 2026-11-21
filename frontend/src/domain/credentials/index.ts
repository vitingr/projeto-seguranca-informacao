// The backend's login and password rules (002 D-14, 003 D-7).

export const LOGIN_PATTERN = /^[a-z0-9._-]{3,64}$/

/** Password length as the backend counts it: code points after NFKC. */
export const passwordLength = (value: string) =>
  [...value.normalize('NFKC')].length
