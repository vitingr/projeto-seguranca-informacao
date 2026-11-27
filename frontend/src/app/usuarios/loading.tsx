import { Skeleton } from '@/components/ui/shadcn/skeleton'

const Loading = () => (
  <main
    aria-busy="true"
    aria-label="Carregando"
    className="mx-auto flex w-full max-w-7xl flex-col gap-8 px-4 py-10 sm:px-6 lg:px-8"
  >
    <div className="flex flex-col gap-3">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-4 w-full max-w-md" />
    </div>
    <div className="grid gap-6 lg:grid-cols-10">
      <Skeleton className="h-56 lg:col-span-7" />
      <div className="flex flex-col gap-4 lg:col-span-3">
        <Skeleton className="h-32" />
        <Skeleton className="h-24" />
      </div>
    </div>
    <div className="flex flex-col gap-4">
      <Skeleton className="h-6 w-40" />
      <Skeleton className="h-8 w-full max-w-lg" />
      <div className="border-border flex flex-col divide-y rounded-sm border">
        {[0, 1, 2, 3, 4].map(row => (
          <div className="flex items-center gap-4 px-3 py-3" key={row}>
            <Skeleton className="h-4 flex-1" />
            <Skeleton className="hidden h-4 w-24 md:block" />
            <Skeleton className="h-5 w-20" />
            <Skeleton className="hidden h-4 w-28 md:block" />
          </div>
        ))}
      </div>
    </div>
  </main>
)

export default Loading
