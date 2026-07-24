import type { ExecutionBinding } from "../src/types.js";

export const TEST_EXECUTION_BINDING: ExecutionBinding = Object.freeze({
  schema_version: 1,
  run_id: "run-00000000000000000000000000000000",
  attempt_id: "attempt-test",
  manifest_sha256: "a".repeat(64),
  workspace_fingerprint: "b".repeat(64),
  execution_owner: "typescript-host",
});
