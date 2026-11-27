/** A JSON object of an answer, before it is checked. */
export type Fields = Record<string, unknown>

/** Which answer shape: the `comum` user's (`own`) or the admin's (`queue`). */
export type Scope = 'own' | 'queue'
