import { createNarrator } from "./src/narrator.js";

const narrator = createNarrator({ cwd: "/Users/callumcowie/repos/Aether", outputMode: "visual" });

const originalWrite = process.stdout.write;
const captured: string[] = [];
process.stdout.write = ((chunk: any, encoding?: any, cb?: any) => {
  const str = typeof chunk === "string" ? chunk : chunk.toString();
  captured.push(str);
  if (typeof cb === "function") cb();
  return true;
}) as any;

narrator.onEvent({ topic: "ceremony.chamber.seal", payload: {} });
narrator.onEvent({ topic: "ceremony.build.spawn", payload: { caste: "sage", name: "Sage-1", task: "Review" } });

process.stdout.write = originalWrite;
console.log("CAPTURED:", JSON.stringify(captured));
console.log("JOINED:", captured.join(""));
