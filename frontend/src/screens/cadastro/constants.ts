// What each role can do, shown next to the form to help pick one.
export const ROLE_GUIDE = [
  {
    items: [
      'Publica posts no mural',
      'Edita e exclui os próprios posts',
      'Acessa a própria conta'
    ],
    name: 'Usuário comum'
  },
  {
    items: [
      'Cadastra usuários',
      'Remove posts do mural, com motivo',
      'Consulta a auditoria'
    ],
    name: 'Administrador'
  }
] as const
