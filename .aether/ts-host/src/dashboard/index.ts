/**
 * Dashboard barrel exports.
 *
 * Re-exports the main dashboard factory and all sub-module types
 * so consumers can import from a single entry point.
 */

export { createDashboard, type Dashboard, type DashboardOptions } from "../dashboard.js";
export {
  createWorkerWidget,
  renderWorkerWidget,
  updateWorkerWidget,
  formatDuration,
  type WorkerWidget,
  type WorkerState,
} from "./worker-widget.js";
export {
  createChamberMap,
  renderChamberMap,
  extractDirectoryPrefix,
  type ChamberMap,
  type ChamberActivity,
} from "./chamber-map.js";
export {
  renderDashboardFrame,
  clearDashboard,
  renderHeader,
  renderFooter,
  type DashboardFrameData,
} from "./dashboard-renderer.js";
