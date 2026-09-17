package colony

import (
	"fmt"
	"strings"
)

// TaskReference identifies a task without changing its authored ID or content.
type TaskReference struct {
	PhaseIndex int
	TaskIndex  int
}

// TaskReferenceIndex gives numeric/runtime IDs and semantic IDs one namespace.
// An alias may name its own task's ID, but never another task. The index owns
// no slices from the plan, so resolving accepted work cannot rewrite approval
// hashes or the active revision projection.
type TaskReferenceIndex struct{ references map[string]TaskReference }

func canonicalTaskReference(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func NewTaskReferenceIndex(phases []Phase) (*TaskReferenceIndex, error) {
	index := &TaskReferenceIndex{references: make(map[string]TaskReference)}
	fields := make(map[string]string)
	for p, phase := range phases {
		for t, task := range phase.Tasks {
			location := TaskReference{PhaseIndex: p, TaskIndex: t}
			aliases := []struct{ value, field string }{{task.SemanticID, "semantic_id"}}
			if task.ID != nil {
				aliases = append(aliases, struct{ value, field string }{*task.ID, "id"})
			}
			for _, raw := range aliases {
				alias := canonicalTaskReference(raw.value)
				if alias == "" {
					continue
				}
				if prior, exists := index.references[alias]; exists && prior != location {
					return nil, fmt.Errorf("ambiguous task reference %q: phases[%d].tasks[%d].%s conflicts with %s", alias, p, t, raw.field, fields[alias])
				}
				index.references[alias] = location
				fields[alias] = fmt.Sprintf("phases[%d].tasks[%d].%s", p, t, raw.field)
			}
		}
	}
	return index, nil
}

func (index *TaskReferenceIndex) Resolve(reference string) (TaskReference, bool) {
	if index == nil {
		return TaskReference{}, false
	}
	location, ok := index.references[canonicalTaskReference(reference)]
	return location, ok
}

func taskDependencyID(task Task) string {
	if task.ID != nil && canonicalTaskReference(*task.ID) != "" {
		return canonicalTaskReference(*task.ID)
	}
	return canonicalTaskReference(task.SemanticID)
}

// ResolveTaskDependencies returns a private scheduling projection. Persisted
// plans keep their exact dependency strings; all graph readers share this
// resolver instead of each inventing numeric-only or semantic-only rules.
func ResolveTaskDependencies(phases []Phase) ([]Phase, error) {
	index, err := NewTaskReferenceIndex(phases)
	if err != nil {
		return nil, err
	}
	resolved := append([]Phase(nil), phases...)
	for p, phase := range phases {
		resolved[p].Tasks = append([]Task(nil), phase.Tasks...)
		for t, task := range phase.Tasks {
			if taskDependencyID(task) == "" {
				continue
			} // Legacy unnamed tasks have no graph identity.
			if task.DependsOn == nil {
				continue
			}
			deps := make([]string, 0, len(task.DependsOn))
			for _, dependency := range task.DependsOn {
				target, ok := index.Resolve(dependency)
				if !ok {
					return nil, &MissingDepError{Task: taskDependencyID(task), MissingDep: dependency}
				}
				deps = append(deps, taskDependencyID(phases[target.PhaseIndex].Tasks[target.TaskIndex]))
			}
			resolved[p].Tasks[t].DependsOn = deps
		}
	}
	return resolved, nil
}
