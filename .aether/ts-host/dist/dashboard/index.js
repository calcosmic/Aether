/**
 * Dashboard barrel exports.
 *
 * Re-exports the main dashboard factory and all sub-module types
 * so consumers can import from a single entry point.
 */
export { createDashboard } from "../dashboard.js";
export { createWorkerWidget, renderWorkerWidget, updateWorkerWidget, formatDuration, } from "./worker-widget.js";
export { createChamberMap, renderChamberMap, extractDirectoryPrefix, } from "./chamber-map.js";
export { renderDashboardFrame, clearDashboard, renderHeader, renderFooter, } from "./dashboard-renderer.js";
