/**
 * FAQ section: SEO-friendly answers on positioning, Cursor/Claude support, and policy.
 */
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import { FAQ_ITEMS } from "@/lib/faq-content"

export function FaqSection() {
  return (
    <section id="faq" className="py-10">
      <div className="mx-auto max-w-[52rem] px-4">
        <h2 className="mb-2 text-xl font-semibold">Frequently asked questions</h2>
        <p className="mb-6 text-muted-foreground">
          MCP guard, vibe coding security tools, AI agent command approval, and
          how SideGuard fits your Cursor or Claude Code workflow.
        </p>
        <Accordion>
          {FAQ_ITEMS.map((item) => (
            <AccordionItem key={item.id} value={item.id}>
              <AccordionTrigger>{item.question}</AccordionTrigger>
              <AccordionContent>
                <p>{item.answer}</p>
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </section>
  )
}
