import { Skeleton } from '@/components/ui/shadcn/skeleton'

export const TicketPageSkeleton = () => (
  <main
    aria-busy="true"
    aria-label="Carregando"
    className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8"
  >
    <Skeleton className="h-8 w-48" />
    <div className="flex flex-col gap-3">
      <Skeleton className="h-4 w-12" />
      <Skeleton className="h-8 w-full max-w-xl" />
      <div className="flex flex-wrap gap-4">
        <Skeleton className="h-5 w-24" />
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-5 w-44" />
      </div>
    </div>
    <Skeleton className="h-9 w-56" />
    <div className="border-border flex flex-col divide-y rounded-sm border">
      {[0, 1, 2, 3].map(row => (
        <div className="flex flex-col gap-2 px-4 py-3 sm:flex-row" key={row}>
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-4 flex-1" />
        </div>
      ))}
    </div>
  </main>
)
