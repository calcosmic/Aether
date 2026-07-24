import { RETIRED_CONTROL_PLANE_MESSAGE } from "./retired.js";

export async function main(): Promise<void> {
  throw new Error(RETIRED_CONTROL_PLANE_MESSAGE);
}

main().catch((err: unknown) => {
  const message = err instanceof Error ? err.message : String(err);
  console.error(message);
  process.exit(1);
});
