import { z } from "zod";

export const EventSchema = z.object({
  type: z.string().min(1),
  timestamp: z.string().datetime({ offset: true }),
  payload: z.record(z.unknown()),
});

export type ColonyEvent = z.infer<typeof EventSchema>;

export function parseEventLine(line: string): ColonyEvent {
  const trimmed = line.trim();
  if (!trimmed) {
    throw new Error("Empty event line");
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch (err) {
    const context = trimmed.length > 80 ? trimmed.slice(0, 80) + "..." : trimmed;
    throw new Error(
      `Invalid JSON in event line: ${err instanceof Error ? err.message : String(err)} (line: ${context})`
    );
  }
  return EventSchema.parse(parsed);
}
