import type {
  Platform,
  PlatformAdapter,
  DispatchParams,
  DispatchResult,
  AvailabilityStatus,
  HealthStatus,
} from "./types.js";
import { RETIRED_CONTROL_PLANE_MESSAGE } from "../retired.js";

export class CodexAdapter implements PlatformAdapter {
  readonly platform: Platform = "codex";

  async dispatch(_params: DispatchParams): Promise<DispatchResult> {
    throw new Error(RETIRED_CONTROL_PLANE_MESSAGE);
  }

  async preflight(): Promise<AvailabilityStatus> {
    return {
      platform: this.platform,
      available: false,
      category: "provider_config_invalid",
      reason: RETIRED_CONTROL_PLANE_MESSAGE,
    };
  }

  async health(): Promise<HealthStatus> {
    return { status: "unhealthy", message: RETIRED_CONTROL_PLANE_MESSAGE };
  }
}
