import { Skeleton } from '@/components/ui/shadcn/skeleton'

// A quiet skeleton of the builder while the session is checked (spec
// section 7, "Loading"), with the layout of plan 5.4.
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
    <div className="flex items-center gap-4">
      <Skeleton className="size-9 rounded-full" />
      <Skeleton className="h-4 flex-1" />
      <Skeleton className="size-9 rounded-full" />
      <Skeleton className="h-4 flex-1" />
      <Skeleton className="size-9 rounded-full" />
    </div>
    <div className="grid gap-6 lg:grid-cols-10">
      <div className="border-border grid gap-3 rounded-sm border p-4 sm:grid-cols-2 sm:p-6 lg:col-span-7">
        <Skeleton className="h-20" />
        <Skeleton className="h-20" />
        <Skeleton className="h-20" />
        <Skeleton className="h-20" />
      </div>
      <div className="flex flex-col gap-4 lg:col-span-3">
        <Skeleton className="h-32" />
        <Skeleton className="h-24" />
      </div>
    </div>
  </main>
)

export default Loading
