import type {
  AdminTicketDetail,
  AdminTicketSummary,
  AuthorList,
  CadastroRequest,
  ChangeStatusRequest,
  CreateMessageRequest,
  CreateTicketRequest,
  LoginRequest,
  QueuePage,
  StatusChangeResponse,
  ThreadPage,
  TicketDataByType,
  TicketDetail,
  TicketPage,
  TicketResponse,
  TicketSummary,
  UserResponse
} from './types'

export const ERROR_CODES = [
  'bad_request',
  'unauthorized',
  'forbidden',
  'not_found',
  'conflict',
  'unprocessable_entity',
  'too_many_requests',
  'internal'
] as const

export const CSRF_HEADER = 'X-CSRF-Token'

export const LOGIN_REQUEST_FIELDS = [
  'login',
  'senha'
] as const satisfies readonly (keyof LoginRequest)[]

export const CADASTRO_REQUEST_FIELDS = [
  'login',
  'senha',
  'papel'
] as const satisfies readonly (keyof CadastroRequest)[]

export const USER_RESPONSE_FIELDS = [
  'user_id',
  'login',
  'role',
  'created_at'
] as const satisfies readonly (keyof UserResponse)[]

export const TICKET_TYPES = ['acesso', 'bug', 'deploy', 'outro'] as const

export const TICKET_STATUSES = [
  'aberto',
  'em_andamento',
  'resolvido',
  'fechado'
] as const

export const ACCESS_LEVELS = ['leitura', 'escrita', 'administrador'] as const

export const SEVERITIES = ['baixa', 'media', 'alta', 'critica'] as const

export const ENVIRONMENTS = ['dev', 'homolog', 'prod'] as const

export const CREATE_TICKET_REQUEST_FIELDS = [
  'tipo',
  'dados'
] as const satisfies readonly (keyof CreateTicketRequest)[]

export const TICKET_DATA_FIELDS = {
  acesso: ['sistema', 'nivel_acesso', 'justificativa'],
  bug: ['sistema', 'descricao', 'passos', 'severidade'],
  deploy: ['projeto', 'repositorio', 'ambiente', 'observacoes'],
  outro: ['titulo', 'descricao']
} as const satisfies {
  [Type in keyof TicketDataByType]: readonly (keyof TicketDataByType[Type])[]
}

export const TICKET_RESPONSE_FIELDS = [
  'id',
  'type',
  'title',
  'status',
  'created_at',
  'updated_at'
] as const satisfies readonly (keyof TicketResponse)[]

export const TICKET_PAGE_SIZE = 20

export const THREAD_PAGE_SIZE = 100

export const ENTRY_KINDS = ['message', 'status_change'] as const

export const ENTRY_SIDES = ['author', 'support'] as const

export const TICKET_SUMMARY_FIELDS = [
  'id',
  'type',
  'status',
  'title',
  'created_at',
  'updated_at'
] as const satisfies readonly (keyof TicketSummary)[]

export const ADMIN_TICKET_SUMMARY_FIELDS = [
  'id',
  'type',
  'status',
  'title',
  'author_login',
  'created_at',
  'updated_at'
] as const satisfies readonly (keyof AdminTicketSummary)[]

export const TICKET_PAGE_FIELDS = [
  'items',
  'total',
  'page',
  'pages',
  'page_size'
] as const satisfies readonly (keyof TicketPage)[]

export const QUEUE_PAGE_FIELDS = [
  ...TICKET_PAGE_FIELDS,
  'status_counts'
] as const satisfies readonly (keyof QueuePage)[]

export const AUTHOR_LIST_FIELDS = [
  'logins'
] as const satisfies readonly (keyof AuthorList)[]

export const TICKET_DETAIL_FIELDS = [
  'id',
  'type',
  'status',
  'title',
  'created_at',
  'updated_at',
  'data'
] as const satisfies readonly (keyof TicketDetail)[]

export const ADMIN_TICKET_DETAIL_FIELDS = [
  'id',
  'type',
  'status',
  'title',
  'author_login',
  'created_at',
  'updated_at',
  'data'
] as const satisfies readonly (keyof AdminTicketDetail)[]

export const THREAD_ENTRY_FIELDS = [
  'id',
  'kind',
  'side',
  'mine',
  'text',
  'from',
  'to',
  'created_at'
] as const

export const ADMIN_THREAD_ENTRY_FIELDS = [
  'id',
  'kind',
  'side',
  'mine',
  'actor_login',
  'text',
  'from',
  'to',
  'created_at'
] as const

export const THREAD_PAGE_FIELDS = [
  'entries',
  'has_more'
] as const satisfies readonly (keyof ThreadPage)[]

export const CREATE_MESSAGE_REQUEST_FIELDS = [
  'texto'
] as const satisfies readonly (keyof CreateMessageRequest)[]

export const CHANGE_STATUS_REQUEST_FIELDS = [
  'de',
  'para'
] as const satisfies readonly (keyof ChangeStatusRequest)[]

export const STATUS_CHANGE_RESPONSE_FIELDS = [
  'id',
  'status',
  'updated_at'
] as const satisfies readonly (keyof StatusChangeResponse)[]
