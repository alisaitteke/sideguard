/**
 * 404 page: unknown routes link back to home.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
import { Link } from "react-router-dom"

export function NotFoundPage() {
  return (
    <main>
      <header className="py-16 text-center sm:py-24">
        <div className="mx-auto max-w-[52rem] px-4">
          <div className="mb-6 flex justify-center">
            <img
              src="/assets/logo.svg"
              alt="SideGuard"
              width={48}
              height={48}
              className="brightness-0 invert"
            />
          </div>
          <h1 className="text-2xl font-semibold sm:text-3xl">Page not found</h1>
          <p className="mt-3 text-muted-foreground">
            The page you requested does not exist.
          </p>
          <p className="mt-6">
            <Link to="/" className="text-primary hover:underline">
              ← Back to SideGuard home
            </Link>
          </p>
        </div>
      </header>
    </main>
  )
}
