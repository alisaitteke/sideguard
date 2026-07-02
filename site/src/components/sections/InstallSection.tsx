/**
 * Install command block with clipboard copy and GitHub raw fallback accordion.
 * Plan: docs/plans/2026-07-02-1705-vite-shadcn-site/vss-phase-2.0-landing-router.md
 */
import { toast } from "sonner"

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import { Button } from "@/components/ui/button"
import {
  FALLBACK_INSTALL_COMMAND,
  INSTALL_COMMAND,
} from "@/lib/install"

async function copyInstallCommand() {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(INSTALL_COMMAND)
      toast.success("Copied!")
    } else {
      toast.error("Select and copy manually")
    }
  } catch {
    toast.error("Select and copy manually")
  }
}

export function InstallSection() {
  return (
    <section id="install" className="py-10">
      <div className="mx-auto max-w-[52rem] px-4">
        <h2 className="mb-6 text-xl font-semibold">Install</h2>
        <div className="space-y-4">
          <div className="overflow-x-auto rounded-lg border border-border bg-card p-4">
            <pre className="font-mono text-sm leading-relaxed text-foreground">
              <code id="install-command">{INSTALL_COMMAND}</code>
            </pre>
            <div className="mt-4">
              <Button
                type="button"
                id="copy-install"
                onClick={() => void copyInstallCommand()}
              >
                Copy command
              </Button>
            </div>
          </div>

          <Accordion>
            <AccordionItem value="fallback">
              <AccordionTrigger>GitHub raw fallback</AccordionTrigger>
              <AccordionContent>
                <pre className="overflow-x-auto rounded-lg border border-border bg-muted/50 p-4 font-mono text-sm">
                  <code>{FALLBACK_INSTALL_COMMAND}</code>
                </pre>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
      </div>
    </section>
  )
}
