import { Skeleton } from '@/components/ui/shadcn/skeleton'

// A quiet skeleton of the builder while the session is checked (spec
// section 7, "Loading").
const Loading = () => (
  <main
    aria-busy="true"
    aria-label="Carregando"
    className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-4 py-10"
  >
    <div className="flex flex-col gap-3">
      <Skeleton className="h-7 w-40" />
      <Skeleton className="h-4 w-full max-w-md" />
    </div>
    <div className="grid gap-8 md:grid-cols-4">
      <div className="flex gap-5 md:flex-col md:gap-3">
        <Skeleton className="h-4 w-16" />
        <Skeleton className="h-4 w-20" />
        <Skeleton className="h-4 w-20" />
      </div>
      <div className="grid gap-3 sm:grid-cols-2 md:col-span-3">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
    </div>
  </main>
)

export default Loading
