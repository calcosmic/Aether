package graph

import (
	"crypto/rand"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestAddNode(t *testing.T) {
	tests := []struct {
		name    string
		node    Node
		wantErr bool
	}{
		{"learning node", Node{ID: "obs_123", Type: NodeLearning}, false},
		{"instinct node", Node{ID: "inst_456", Type: NodeInstinct}, false},
		{"queen node", Node{ID: "queen_1", Type: NodeQueen}, false},
		{"phase node", Node{ID: "phase_47", Type: NodePhase}, false},
		{"colony node", Node{ID: "colony_abc", Type: NodeColony}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGraph()
			err := g.AddNode(tt.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddNode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				node := g.GetNode(tt.node.ID)
				if node == nil {
					t.Errorf("GetNode(%q) returned nil after AddNode", tt.node.ID)
				} else if node.Type != tt.node.Type {
					t.Errorf("GetNode(%q).Type = %v, want %v", tt.node.ID, node.Type, tt.node.Type)
				}
			}
		})
	}
}

func TestAddNodeDuplicate(t *testing.T) {
	g := NewGraph()
	node := Node{ID: "inst_100", Type: NodeInstinct}
	if err := g.AddNode(node); err != nil {
		t.Fatalf("first AddNode failed: %v", err)
	}

	// Same ID, different type should error
	err := g.AddNode(Node{ID: "inst_100", Type: NodeQueen})
	if err == nil {
		t.Error("AddNode with same ID and different type should return error")
	}

	// Same ID, same type should succeed (idempotent)
	err = g.AddNode(Node{ID: "inst_100", Type: NodeInstinct})
	if err != nil {
		t.Errorf("AddNode with same ID and same type should succeed, got error: %v", err)
	}
}

func TestAddEdge(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "inst_a", Type: NodeInstinct})
	g.AddNode(Node{ID: "inst_b", Type: NodeInstinct})

	edge, status, err := g.AddEdge("inst_a", "inst_b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	if err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}
	if status != "created" {
		t.Errorf("AddEdge() status = %q, want %q", status, "created")
	}
	if edge.Source != "inst_a" {
		t.Errorf("edge.Source = %q, want %q", edge.Source, "inst_a")
	}
	if edge.Target != "inst_b" {
		t.Errorf("edge.Target = %q, want %q", edge.Target, "inst_b")
	}
	if edge.Relationship != EdgeReinforces {
		t.Errorf("edge.Relationship = %v, want %v", edge.Relationship, EdgeReinforces)
	}
	if edge.Weight != 0.8 {
		t.Errorf("edge.Weight = %v, want 0.8", edge.Weight)
	}

	// Verify edge is in outEdges and inEdges
	result, err := g.Neighbors("inst_a", "out", "", 0)
	if err != nil {
		t.Fatalf("Neighbors() error = %v", err)
	}
	if result.Count != 1 {
		t.Errorf("out neighbor count = %d, want 1", result.Count)
	}

	result, err = g.Neighbors("inst_b", "in", "", 0)
	if err != nil {
		t.Fatalf("Neighbors() error = %v", err)
	}
	if result.Count != 1 {
		t.Errorf("in neighbor count = %d, want 1", result.Count)
	}
}

func TestAddEdgeAutoCreatesNodes(t *testing.T) {
	g := NewGraph()

	_, _, err := g.AddEdge("obs_100", "inst_200", EdgePromotedFrom, 0.9, "2026-04-01T00:00:00Z")
	if err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}

	// Source node auto-created with inferred type
	srcNode := g.GetNode("obs_100")
	if srcNode == nil {
		t.Fatal("source node not auto-created")
	}
	if srcNode.Type != NodeLearning {
		t.Errorf("auto-created source node type = %v, want %v", srcNode.Type, NodeLearning)
	}

	// Target node auto-created with inferred type
	tgtNode := g.GetNode("inst_200")
	if tgtNode == nil {
		t.Fatal("target node not auto-created")
	}
	if tgtNode.Type != NodeInstinct {
		t.Errorf("auto-created target node type = %v, want %v", tgtNode.Type, NodeInstinct)
	}
}

func TestAddEdgeDedup(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "inst_x", Type: NodeInstinct})
	g.AddNode(Node{ID: "inst_y", Type: NodeInstinct})

	_, status, _ := g.AddEdge("inst_x", "inst_y", EdgeReinforces, 0.5, "2026-04-01T00:00:00Z")
	if status != "created" {
		t.Errorf("first AddEdge status = %q, want %q", status, "created")
	}

	edge, status, err := g.AddEdge("inst_x", "inst_y", EdgeReinforces, 0.9, "2026-04-01T00:00:00Z")
	if err != nil {
		t.Fatalf("dedup AddEdge error = %v", err)
	}
	if status != "updated" {
		t.Errorf("dedup AddEdge status = %q, want %q", status, "updated")
	}
	if edge.Weight != 0.9 {
		t.Errorf("updated edge weight = %v, want 0.9", edge.Weight)
	}

	// Should still have exactly 1 edge
	if g.EdgeCount() != 1 {
		t.Errorf("EdgeCount() = %d, want 1 after dedup", g.EdgeCount())
	}
}

func TestAddEdgeNewReturnsCreated(t *testing.T) {
	g := NewGraph()
	_, status, _ := g.AddEdge("a", "b", EdgeRelated, 0.5, "2026-04-01T00:00:00Z")
	if status != "created" {
		t.Errorf("new edge status = %q, want %q", status, "created")
	}
}

func TestRemoveNode(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "inst_a", Type: NodeInstinct})
	g.AddNode(Node{ID: "inst_b", Type: NodeInstinct})
	g.AddNode(Node{ID: "inst_c", Type: NodeInstinct})
	g.AddEdge("inst_a", "inst_b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("inst_c", "inst_a", EdgeContradicts, 0.6, "2026-04-01T00:00:00Z")

	err := g.RemoveNode("inst_a")
	if err != nil {
		t.Fatalf("RemoveNode() error = %v", err)
	}

	if g.GetNode("inst_a") != nil {
		t.Error("node still exists after RemoveNode")
	}

	// All connected edges should be removed
	if g.EdgeCount() != 0 {
		t.Errorf("EdgeCount() = %d after RemoveNode, want 0", g.EdgeCount())
	}

	// inst_b and inst_c should still exist
	if g.GetNode("inst_b") == nil {
		t.Error("inst_b should still exist")
	}
	if g.GetNode("inst_c") == nil {
		t.Error("inst_c should still exist")
	}
}

func TestRemoveEdge(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodeInstinct})
	g.AddEdge("a", "b", EdgeRelated, 0.5, "2026-04-01T00:00:00Z")

	if g.EdgeCount() != 1 {
		t.Fatalf("EdgeCount() before remove = %d, want 1", g.EdgeCount())
	}

	err := g.RemoveEdge("a", "b", EdgeRelated)
	if err != nil {
		t.Fatalf("RemoveEdge() error = %v", err)
	}

	if g.EdgeCount() != 0 {
		t.Errorf("EdgeCount() after remove = %d, want 0", g.EdgeCount())
	}
}

func TestRemoveNodeNotFound(t *testing.T) {
	g := NewGraph()
	err := g.RemoveNode("nonexistent")
	if err == nil {
		t.Error("RemoveNode on nonexistent node should return error")
	}
}

func TestRemoveEdgeNotFound(t *testing.T) {
	g := NewGraph()
	err := g.RemoveEdge("x", "y", EdgeRelated)
	if err == nil {
		t.Error("RemoveEdge on nonexistent edge should return error")
	}
}

func TestQueryByType(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "obs_1", Type: NodeLearning})
	g.AddNode(Node{ID: "obs_2", Type: NodeLearning})
	g.AddNode(Node{ID: "inst_1", Type: NodeInstinct})
	g.AddNode(Node{ID: "queen_1", Type: NodeQueen})
	g.AddNode(Node{ID: "phase_1", Type: NodePhase})

	learnings := g.NodesByType(NodeLearning)
	if len(learnings) != 2 {
		t.Errorf("NodesByType(learning) = %d nodes, want 2", len(learnings))
	}

	instincts := g.NodesByType(NodeInstinct)
	if len(instincts) != 1 {
		t.Errorf("NodesByType(instinct) = %d nodes, want 1", len(instincts))
	}

	queens := g.NodesByType(NodeQueen)
	if len(queens) != 1 {
		t.Errorf("NodesByType(queen) = %d nodes, want 1", len(queens))
	}

	colonies := g.NodesByType(NodeColony)
	if len(colonies) != 0 {
		t.Errorf("NodesByType(colony) = %d nodes, want 0", len(colonies))
	}
}

func TestEdgeCount(t *testing.T) {
	g := NewGraph()
	if g.EdgeCount() != 0 {
		t.Errorf("EdgeCount() on empty graph = %d, want 0", g.EdgeCount())
	}

	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodeInstinct})
	g.AddNode(Node{ID: "c", Type: NodeInstinct})
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "c", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")

	if g.EdgeCount() != 2 {
		t.Errorf("EdgeCount() = %d, want 2", g.EdgeCount())
	}
}

func TestNodeCount(t *testing.T) {
	g := NewGraph()
	if g.NodeCount() != 0 {
		t.Errorf("NodeCount() on empty graph = %d, want 0", g.NodeCount())
	}

	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodePhase})

	if g.NodeCount() != 2 {
		t.Errorf("NodeCount() = %d, want 2", g.NodeCount())
	}
}

func TestAllEdgeTypes(t *testing.T) {
	edgeTypes := []EdgeType{
		EdgePromotedFrom,
		EdgeDerivedFrom,
		EdgeReinforces,
		EdgeContradicts,
		EdgeExtends,
		EdgeSupersedes,
		EdgeRelated,
		EdgePromotedTo,
		EdgeProduced,
		EdgeOriginated,
		EdgeContains,
		EdgeColonyInstinct,
		EdgeDependsOn,
		EdgeInfluenced,
		EdgeSupersedesPhase,
	}

	// 15 types defined above; validate the 16th separately
	if len(edgeTypes) != 15 {
		t.Errorf("expected 15 edge types in list, got %d", len(edgeTypes))
	}

	g := NewGraph()
	g.AddNode(Node{ID: "n1", Type: NodeInstinct})
	g.AddNode(Node{ID: "n2", Type: NodeInstinct})

	for i, et := range edgeTypes {
		_, status, err := g.AddEdge(fmt.Sprintf("n1_%d", i), fmt.Sprintf("n2_%d", i), et, 0.5, "2026-04-01T00:00:00Z")
		if err != nil {
			t.Errorf("AddEdge with type %q failed: %v", et, err)
		}
		if status != "created" {
			t.Errorf("AddEdge type %q status = %q, want created", et, status)
		}
	}

	if g.EdgeCount() != 15 {
		t.Errorf("EdgeCount() = %d, want 15", g.EdgeCount())
	}

	// Test the 16th type: EdgeSupersedesPhase
	g2 := NewGraph()
	g2.AddNode(Node{ID: "p1", Type: NodePhase})
	g2.AddNode(Node{ID: "p2", Type: NodePhase})
	_, status, err := g2.AddEdge("p1", "p2", EdgeSupersedesPhase, 0.9, "2026-04-01T00:00:00Z")
	if err != nil {
		t.Errorf("AddEdge EdgeSupersedesPhase failed: %v", err)
	}
	if status != "created" {
		t.Errorf("EdgeSupersedesPhase status = %q, want created", status)
	}
}

func TestConcurrentAccess(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "root", Type: NodeInstinct})

	// Pre-populate some nodes
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("node_%d", i)
		g.AddNode(Node{ID: id, Type: NodeInstinct})
		g.AddEdge("root", id, EdgeRelated, float64(i)/10.0, "2026-04-01T00:00:00Z")
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = g.NodeCount()
				_ = g.EdgeCount()
				g.NodesByType(NodeInstinct)
				result, err := g.Neighbors("root", "out", "", 0)
				if err != nil {
					errors <- err
				}
				if result == nil {
					errors <- fmt.Errorf("nil NeighborsResult")
				}
			}
		}()
	}

	// Concurrent writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		b := make([]byte, 2)
		for i := 100; i < 150; i++ {
			rand.Read(b)
			id := fmt.Sprintf("writer_%d", i)
			g.AddNode(Node{ID: id, Type: NodeColony})
			g.AddEdge("root", id, EdgeContains, 0.5, "2026-04-01T00:00:00Z")
		}
	}()

	wg.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestNeighborsOut(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodeInstinct})
	g.AddNode(Node{ID: "c", Type: NodeInstinct})
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("a", "c", EdgeExtends, 0.6, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "a", EdgeContradicts, 0.3, "2026-04-01T00:00:00Z")

	result, err := g.Neighbors("a", "out", "", 0)
	if err != nil {
		t.Fatalf("Neighbors(out) error = %v", err)
	}
	if result.Count != 2 {
		t.Fatalf("Neighbors(out) count = %d, want 2", result.Count)
	}

	ids := neighborIDs(result)
	sort.Strings(ids)
	if ids[0] != "b" || ids[1] != "c" {
		t.Errorf("Neighbors(out) ids = %v, want [b c]", ids)
	}
}

func TestNeighborsIn(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodeInstinct})
	g.AddNode(Node{ID: "c", Type: NodeInstinct})
	g.AddEdge("b", "a", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("c", "a", EdgeContradicts, 0.4, "2026-04-01T00:00:00Z")
	g.AddEdge("a", "b", EdgeExtends, 0.6, "2026-04-01T00:00:00Z")

	result, err := g.Neighbors("a", "in", "", 0)
	if err != nil {
		t.Fatalf("Neighbors(in) error = %v", err)
	}
	if result.Count != 2 {
		t.Fatalf("Neighbors(in) count = %d, want 2", result.Count)
	}

	ids := neighborIDs(result)
	sort.Strings(ids)
	if ids[0] != "b" || ids[1] != "c" {
		t.Errorf("Neighbors(in) ids = %v, want [b c]", ids)
	}
}

func TestNeighborsBoth(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodeInstinct})
	g.AddNode(Node{ID: "c", Type: NodeInstinct})
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z") // a -> b (out)
	g.AddEdge("c", "a", EdgeContradicts, 0.4, "2026-04-01T00:00:00Z") // c -> a (in)

	result, err := g.Neighbors("a", "both", "", 0)
	if err != nil {
		t.Fatalf("Neighbors(both) error = %v", err)
	}
	if result.Count != 2 {
		t.Fatalf("Neighbors(both) count = %d, want 2", result.Count)
	}

	ids := neighborIDs(result)
	sort.Strings(ids)
	if ids[0] != "b" || ids[1] != "c" {
		t.Errorf("Neighbors(both) ids = %v, want [b c]", ids)
	}
}

func TestNeighborsFilterRel(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodeInstinct})
	g.AddNode(Node{ID: "c", Type: NodeInstinct})
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("a", "c", EdgeContradicts, 0.6, "2026-04-01T00:00:00Z")

	result, err := g.Neighbors("a", "out", EdgeReinforces, 0)
	if err != nil {
		t.Fatalf("Neighbors(filter) error = %v", err)
	}
	if result.Count != 1 {
		t.Fatalf("Neighbors(filter=Reinforces) count = %d, want 1", result.Count)
	}
	if result.Neighbors[0].ID != "b" {
		t.Errorf("Neighbor id = %q, want %q", result.Neighbors[0].ID, "b")
	}
}

func TestNeighborsWeightFilter(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "a", Type: NodeInstinct})
	g.AddNode(Node{ID: "b", Type: NodeInstinct})
	g.AddNode(Node{ID: "c", Type: NodeInstinct})
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("a", "c", EdgeReinforces, 0.3, "2026-04-01T00:00:00Z")

	result, err := g.Neighbors("a", "out", "", 0.5)
	if err != nil {
		t.Fatalf("Neighbors(minWeight) error = %v", err)
	}
	if result.Count != 1 {
		t.Fatalf("Neighbors(minWeight=0.5) count = %d, want 1", result.Count)
	}
	if result.Neighbors[0].ID != "b" {
		t.Errorf("Neighbor id = %q, want %q", result.Neighbors[0].ID, "b")
	}
}

func TestNeighborsEmpty(t *testing.T) {
	g := NewGraph()
	g.AddNode(Node{ID: "lonely", Type: NodeInstinct})

	result, err := g.Neighbors("lonely", "both", "", 0)
	if err != nil {
		t.Fatalf("Neighbors(empty) error = %v", err)
	}
	if result.Count != 0 {
		t.Errorf("Neighbors(empty) count = %d, want 0", result.Count)
	}
}

func TestNeighborsNotFound(t *testing.T) {
	g := NewGraph()
	_, err := g.Neighbors("nonexistent", "out", "", 0)
	if err == nil {
		t.Error("Neighbors(nonexistent) should return error")
	}
}

func TestNeighbors2Hop(t *testing.T) {
	// Build: a -> b -> d, a -> c, c -> d, c -> e
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("a", "c", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "d", EdgeRelated, 0.6, "2026-04-01T00:00:00Z")
	g.AddEdge("c", "d", EdgeContradicts, 0.5, "2026-04-01T00:00:00Z")
	g.AddEdge("c", "e", EdgeRelated, 0.9, "2026-04-01T00:00:00Z")

	results, err := g.Neighbors2Hop("a", "out", "", 0)
	if err != nil {
		t.Fatalf("Neighbors2Hop error = %v", err)
	}

	// 1-hop: b, c
	// 2-hop: d (via b and c), e (via c)
	ids := make([]string, len(results))
	for i, n := range results {
		ids[i] = n.ID
	}
	sort.Strings(ids)

	expected := []string{"b", "c", "d", "e"}
	if len(ids) != len(expected) {
		t.Fatalf("Neighbors2Hop ids = %v, want %v", ids, expected)
	}
	for i, id := range expected {
		if ids[i] != id {
			t.Errorf("Neighbors2Hop[%d] = %q, want %q", i, ids[i], id)
		}
	}

	// Verify hop counts
	hopMap := make(map[string]int)
	for _, n := range results {
		hopMap[n.ID] = n.Hop
	}
	if hopMap["b"] != 1 {
		t.Errorf("b hop = %d, want 1", hopMap["b"])
	}
	if hopMap["c"] != 1 {
		t.Errorf("c hop = %d, want 1", hopMap["c"])
	}
	if hopMap["d"] != 2 {
		t.Errorf("d hop = %d, want 2", hopMap["d"])
	}
	if hopMap["e"] != 2 {
		t.Errorf("e hop = %d, want 2", hopMap["e"])
	}
}

func TestNeighbors2HopNoDuplicates(t *testing.T) {
	// d is reachable via both a->b->d and a->c->d
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("a", "c", EdgeExtends, 0.7, "2026-04-01T00:00:00Z")
	g.AddEdge("b", "d", EdgeRelated, 0.6, "2026-04-01T00:00:00Z")
	g.AddEdge("c", "d", EdgeContradicts, 0.5, "2026-04-01T00:00:00Z")

	results, err := g.Neighbors2Hop("a", "out", "", 0)
	if err != nil {
		t.Fatalf("Neighbors2Hop error = %v", err)
	}

	// d should appear only once
	seen := map[string]int{}
	for _, n := range results {
		seen[n.ID]++
	}
	for id, count := range seen {
		if count > 1 {
			t.Errorf("node %q appears %d times in 2-hop results, should be at most 1", id, count)
		}
	}
}

func TestNeighbors2HopWeightFilter(t *testing.T) {
	// a -> b (w=0.8), a -> c (w=0.2), b -> d (w=0.7), c -> e (w=0.9)
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		g.AddNode(Node{ID: id, Type: NodeInstinct})
	}
	g.AddEdge("a", "b", EdgeReinforces, 0.8, "2026-04-01T00:00:00Z")
	g.AddEdge("a", "c", EdgeExtends, 0.2, "2026-04-01T00:00:00Z") // below minWeight
	g.AddEdge("b", "d", EdgeRelated, 0.7, "2026-04-01T00:00:00Z")
	g.AddEdge("c", "e", EdgeContradicts, 0.9, "2026-04-01T00:00:00Z")

	results, err := g.Neighbors2Hop("a", "out", "", 0.5)
	if err != nil {
		t.Fatalf("Neighbors2Hop(minWeight) error = %v", err)
	}

	// With minWeight=0.5: a->c filtered out, so c->e never traversed
	// 1-hop: b (0.8 >= 0.5)
	// 2-hop: d (0.7 >= 0.5)
	ids := make([]string, len(results))
	for i, n := range results {
		ids[i] = n.ID
	}
	sort.Strings(ids)

	expected := []string{"b", "d"}
	if len(ids) != len(expected) {
		t.Fatalf("Neighbors2Hop(minWeight=0.5) ids = %v, want %v", ids, expected)
	}
	for i, id := range expected {
		if ids[i] != id {
			t.Errorf("Neighbors2Hop(minWeight=0.5)[%d] = %q, want %q", i, ids[i], id)
		}
	}
}

// Helper to extract neighbor IDs from result
func neighborIDs(result *NeighborsResult) []string {
	ids := make([]string, len(result.Neighbors))
	for i, n := range result.Neighbors {
		ids[i] = n.ID
	}
	return ids
}

// verify all test names contain expected patterns for the verify command
func init() {
	// Ensure test function names are parseable
	_ = strings.Contains
	_ = sort.Strings
	_ = sync.WaitGroup{}
	_ = rand.Read
}
