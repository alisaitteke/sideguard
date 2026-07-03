/**
 * Second hero: prompt-injection threat scenario — GitHub issue with a hidden
 * LLM instruction line; camera pans, zooms, then scans the text left → right.
 */
import { PromptInjectionScene } from "@/components/sections/PromptInjectionScene"

export function PromptInjectionSection() {
  return (
    <section
      id="prompt-injection"
      aria-labelledby="prompt-injection-heading"
      className="border-b border-border bg-background py-16 sm:py-20"
    >
      <div className="mx-auto max-w-[52rem] px-4">
        <div className="mx-auto mb-10 max-w-[38rem] text-center">
          <p className="mb-3 text-sm font-medium tracking-wide text-primary uppercase">
            The threat
          </p>
          <h2
            id="prompt-injection-heading"
            className="text-2xl font-semibold tracking-tight sm:text-3xl"
          >
            Prompt injection hides in places your agent trusts
          </h2>
        </div>

        <PromptInjectionScene mode="interactive" />

        <p className="mx-auto mt-6 max-w-[38rem] text-center text-sm leading-relaxed text-muted-foreground">
          Your agent may treat issue text as context and run what it finds. A
          single disguised line can exfiltrate SSH keys — no exploit chain
          required.
        </p>
      </div>
    </section>
  )
}
