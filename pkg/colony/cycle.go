package colony

import "fmt"

// CycleError is returned when DetectCycles finds a circular dependency chain
// in the task dependency graph. Tasks lists the cycle path, starting and ending
// with the same task ID.
type CycleError struct {
	Tasks []string
}

func (e *CycleError) Error() string {
	return "circular dependency: " + joinTasks(e.Tasks)
}

// MissingDepError is returned when a task depends on a task ID that does not
// exist in any phase.
type MissingDepError struct {
	Task       string
	MissingDep string
}

func (e *MissingDepError) Error() string {
	return fmt.Sprintf("task %s depends on unknown task %s", e.Task, e.MissingDep)
}

const (
	colorWhite = 0 // unvisited
	colorGray  = 1 // in current DFS path
	colorBlack = 2 // fully explored
)

// DetectCycles validates the task dependency graph across all phases.
// It checks for missing dependency references first, then performs a
// three-color DFS to detect cycles.
func DetectCycles(phases []Phase) error {
	// Build adjacency list and known task ID set.
	adj := make(map[string][]string)
	known := make(map[string]bool)

	for _, phase := range phases {
		for _, task := range phase.Tasks {
			if task.ID == nil {
				continue
			}
			id := *task.ID
			known[id] = true
			if len(task.DependsOn) > 0 {
				adj[id] = append(adj[id], task.DependsOn...)
			}
		}
	}

	// First pass: validate all DependsOn references exist.
	for _, phase := range phases {
		for _, task := range phase.Tasks {
			if task.ID == nil {
				continue
			}
			for _, dep := range task.DependsOn {
				if !known[dep] {
					return &MissingDepError{Task: *task.ID, MissingDep: dep}
				}
			}
		}
	}

	// Second pass: three-color DFS cycle detection.
	color := make(map[string]int)
	path := make([]string, 0, len(known))

	var dfs func(node string) error
	dfs = func(node string) error {
		color[node] = colorGray
		path = append(path, node)

		for _, neighbor := range adj[node] {
			if color[neighbor] == colorGray {
				// Found a cycle: extract it from the path.
				cycle := extractCycle(path, neighbor)
				return &CycleError{Tasks: cycle}
			}
			if color[neighbor] == colorWhite {
				if err := dfs(neighbor); err != nil {
					return err
				}
			}
		}

		path = path[:len(path)-1]
		color[node] = colorBlack
		return nil
	}

	for id := range known {
		if color[id] == colorWhite {
			if err := dfs(id); err != nil {
				return err
			}
		}
	}

	return nil
}

// extractCycle extracts the cycle from the DFS path when a back-edge to
// target is found. It returns the cycle path including target at both ends.
func extractCycle(path []string, target string) []string {
	for i, node := range path {
		if node == target {
			cycle := make([]string, len(path)-i+1)
			copy(cycle, path[i:])
			cycle[len(cycle)-1] = target
			return cycle
		}
	}
	// Should not happen if called correctly.
	return append(path, target)
}

// joinTasks formats a cycle as "a -> b -> c".
func joinTasks(tasks []string) string {
	if len(tasks) == 0 {
		return ""
	}
	result := tasks[0]
	for i := 1; i < len(tasks); i++ {
		result += " -> " + tasks[i]
	}
	return result
}
