# Planner Module Detailed Design

**Date**: 2025-12-15
**Status**: Draft
**Related**: [Architecture Design](./2025-12-14-media-pipeline-architecture-design.md), [Schemas Design](./schemas-detailed-design.md)

---

## Overview

The Planner module is the core of the compilation pipeline. It transforms a validated JobSpec into an executable ProcessingPlan by:

1. Building a Directed Acyclic Graph (DAG) of operations
2. Detecting circular dependencies
3. Performing topological sort to determine execution order
4. Estimating resource requirements (CPU, memory, disk)
5. Optimizing the execution plan
6. Generating metadata for each node

---

## Architecture

### Planner Interface

```go
package planner

type Planner interface {
    // Plan transforms a validated JobSpec into a ProcessingPlan
    Plan(ctx context.Context, spec *schemas.JobSpec) (*schemas.ProcessingPlan, error)
}

type DefaultPlanner struct {
    operatorRegistry OperatorRegistry
    mediaProber      MediaProber
    estimator        ResourceEstimator
    optimizer        PlanOptimizer
}

func NewPlanner(opts ...PlannerOption) *DefaultPlanner {
    // Initialize with defaults
}
```

### Planning Pipeline

```
JobSpec
  ↓
[1. Build DAG] - Create nodes and edges from operations
  ↓
[2. Validate DAG] - Check for cycles, unreachable nodes
  ↓
[3. Topological Sort] - Determine execution order
  ↓
[4. Probe Media] - Detect input media properties (async)
  ↓
[5. Propagate Metadata] - Flow media info through the graph
  ↓
[6. Estimate Resources] - Calculate CPU, memory, disk requirements
  ↓
[7. Optimize Plan] - Merge operations, reduce intermediate files
  ↓
ProcessingPlan
```

---

## Phase 1: Build DAG

### Algorithm

```go
func (p *DefaultPlanner) buildDAG(spec *schemas.JobSpec) (*Graph, error) {
    graph := NewGraph()

    // Step 1: Create input nodes
    for _, input := range spec.Inputs {
        node := &schemas.PlanNode{
            ID:        "input_" + input.ID,
            Type:      "input",
            InputID:   input.ID,
            SourceURI: input.Source,
        }
        graph.AddNode(node)
    }

    // Step 2: Create operation nodes
    for i, op := range spec.Operations {
        node := &schemas.PlanNode{
            ID:       fmt.Sprintf("op_%d_%s", i, op.Op),
            Type:     "operation",
            Operator: op.Op,
            Params:   op.Params,
        }
        graph.AddNode(node)

        // Step 3: Create edges (dependencies)
        if op.Input != "" {
            // Single input
            sourceID := p.resolveReference(op.Input, graph)
            if sourceID == "" {
                return nil, fmt.Errorf("operation %s: input '%s' not found", op.Op, op.Input)
            }
            graph.AddEdge(sourceID, node.ID, inferStreamType(op))
        } else if len(op.Inputs) > 0 {
            // Multiple inputs
            for _, inputRef := range op.Inputs {
                sourceID := p.resolveReference(inputRef, graph)
                if sourceID == "" {
                    return nil, fmt.Errorf("operation %s: input '%s' not found", op.Op, inputRef)
                }
                graph.AddEdge(sourceID, node.ID, inferStreamType(op))
            }
        }
    }

    // Step 4: Create output nodes
    for _, output := range spec.Outputs {
        node := &schemas.PlanNode{
            ID:       "output_" + output.ID,
            Type:     "output",
            OutputID: output.ID,
            DestURI:  output.Destination,
        }
        graph.AddNode(node)

        // Link to operation that produces this output
        sourceID := p.resolveReference(output.ID, graph)
        if sourceID == "" {
            return nil, fmt.Errorf("output '%s' references non-existent operation", output.ID)
        }
        graph.AddEdge(sourceID, node.ID, "both")
    }

    return graph, nil
}
```

### Reference Resolution

Operations reference inputs by ID, which can be:
- An input ID (e.g., "video1")
- An upstream operation's output (e.g., "trimmed")

```go
func (p *DefaultPlanner) resolveReference(ref string, graph *Graph) string {
    // Check if it's an input node
    if node := graph.GetNode("input_" + ref); node != nil {
        return node.ID
    }

    // Check if it's an operation output
    for _, node := range graph.Nodes {
        if node.Type == "operation" {
            // Extract output ID from operation
            // (stored in operation metadata)
            if outputID := node.GetOutputID(); outputID == ref {
                return node.ID
            }
        }
    }

    return ""
}
```

### Example

Given this JobSpec:
```json
{
  "inputs": [{"id": "video1", "source": "..."}],
  "operations": [
    {"op": "trim", "input": "video1", "output": "trimmed"},
    {"op": "loudnorm", "input": "trimmed", "output": "normalized"}
  ],
  "outputs": [{"id": "normalized", "destination": "..."}]
}
```

Generated DAG:
```
input_video1 → op_0_trim → op_1_loudnorm → output_normalized
```

---

## Phase 2: Validate DAG

### Cycle Detection

Use DFS-based cycle detection:

```go
func (g *Graph) DetectCycles() error {
    visited := make(map[string]bool)
    recStack := make(map[string]bool)

    for _, node := range g.Nodes {
        if !visited[node.ID] {
            if err := g.dfsCheckCycle(node.ID, visited, recStack); err != nil {
                return err
            }
        }
    }
    return nil
}

func (g *Graph) dfsCheckCycle(nodeID string, visited, recStack map[string]bool) error {
    visited[nodeID] = true
    recStack[nodeID] = true

    for _, edge := range g.GetOutgoingEdges(nodeID) {
        if !visited[edge.To] {
            if err := g.dfsCheckCycle(edge.To, visited, recStack); err != nil {
                return err
            }
        } else if recStack[edge.To] {
            return fmt.Errorf("cycle detected: %s → %s", nodeID, edge.To)
        }
    }

    recStack[nodeID] = false
    return nil
}
```

### Additional Validations

```go
func (p *DefaultPlanner) validateDAG(graph *Graph) error {
    // 1. Check for cycles
    if err := graph.DetectCycles(); err != nil {
        return err
    }

    // 2. Check for unreachable nodes (nodes with no path from inputs)
    reachable := graph.GetReachableFrom(graph.GetInputNodes())
    if len(reachable) != len(graph.Nodes) {
        return fmt.Errorf("graph contains unreachable nodes")
    }

    // 3. Check for dangling nodes (nodes with no path to outputs)
    reverseReachable := graph.GetReachableBackwardFrom(graph.GetOutputNodes())
    if len(reverseReachable) != len(graph.Nodes) {
        return fmt.Errorf("graph contains dangling nodes")
    }

    // 4. Check operator compatibility (e.g., video-only op can't take audio-only input)
    for _, node := range graph.Nodes {
        if node.Type == "operation" {
            if err := p.checkOperatorCompatibility(node, graph); err != nil {
                return err
            }
        }
    }

    return nil
}
```

---

## Phase 3: Topological Sort

Determine execution order using Kahn's algorithm:

```go
func (g *Graph) TopologicalSort() ([]string, error) {
    // Count incoming edges for each node
    inDegree := make(map[string]int)
    for _, node := range g.Nodes {
        inDegree[node.ID] = 0
    }
    for _, edge := range g.Edges {
        inDegree[edge.To]++
    }

    // Queue of nodes with no incoming edges
    queue := []string{}
    for nodeID, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, nodeID)
        }
    }

    // Process nodes
    result := []string{}
    for len(queue) > 0 {
        nodeID := queue[0]
        queue = queue[1:]
        result = append(result, nodeID)

        // Reduce in-degree of neighbors
        for _, edge := range g.GetOutgoingEdges(nodeID) {
            inDegree[edge.To]--
            if inDegree[edge.To] == 0 {
                queue = append(queue, edge.To)
            }
        }
    }

    if len(result) != len(g.Nodes) {
        return nil, fmt.Errorf("graph contains cycle (detected during topological sort)")
    }

    return result, nil
}
```

### Execution Stages

Group nodes into stages for parallel execution:

```go
func (g *Graph) ComputeExecutionStages() ([][]string, error) {
    inDegree := make(map[string]int)
    for _, node := range g.Nodes {
        inDegree[node.ID] = len(g.GetIncomingEdges(node.ID))
    }

    stages := [][]string{}
    processed := make(map[string]bool)

    for len(processed) < len(g.Nodes) {
        stage := []string{}

        // Find all nodes with in-degree 0 (no unprocessed dependencies)
        for nodeID, degree := range inDegree {
            if !processed[nodeID] && degree == 0 {
                stage = append(stage, nodeID)
            }
        }

        if len(stage) == 0 {
            return nil, fmt.Errorf("cannot compute stages (possible cycle)")
        }

        stages = append(stages, stage)

        // Mark as processed and update in-degrees
        for _, nodeID := range stage {
            processed[nodeID] = true
            for _, edge := range g.GetOutgoingEdges(nodeID) {
                inDegree[edge.To]--
            }
        }
    }

    return stages, nil
}
```

Example stages:
```
Stage 0: [input_video1, input_bgm]
Stage 1: [op_0_trim]
Stage 2: [op_1_loudnorm, op_2_trim_audio]
Stage 3: [op_3_mix]
Stage 4: [output_final]
```

Nodes in the same stage can be processed in parallel (for future optimization).

---

## Phase 4: Probe Media

Detect input media properties asynchronously:

```go
type MediaProber interface {
    Probe(ctx context.Context, uri string) (*schemas.MediaInfo, error)
}

type FFprobeProber struct {
    timeout time.Duration
}

func (p *FFprobeProber) Probe(ctx context.Context, uri string) (*schemas.MediaInfo, error) {
    // Download to temp file if remote
    localPath, cleanup, err := p.ensureLocal(ctx, uri)
    if err != nil {
        return nil, err
    }
    defer cleanup()

    // Run ffprobe
    cmd := exec.CommandContext(ctx,
        "ffprobe",
        "-v", "quiet",
        "-print_format", "json",
        "-show_format",
        "-show_streams",
        localPath,
    )

    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("ffprobe failed: %w", err)
    }

    // Parse JSON output
    var result FFprobeResult
    if err := json.Unmarshal(output, &result); err != nil {
        return nil, err
    }

    // Convert to MediaInfo
    return p.parseFFprobeResult(&result), nil
}
```

### Parallel Probing

Probe all inputs concurrently:

```go
func (p *DefaultPlanner) probeInputs(ctx context.Context, graph *Graph) error {
    inputNodes := graph.GetInputNodes()

    // Probe in parallel with limited concurrency
    sem := make(chan struct{}, 5) // Max 5 concurrent probes
    errCh := make(chan error, len(inputNodes))

    for _, node := range inputNodes {
        go func(n *schemas.PlanNode) {
            sem <- struct{}{}
            defer func() { <-sem }()

            mediaInfo, err := p.mediaProber.Probe(ctx, n.SourceURI)
            if err != nil {
                errCh <- fmt.Errorf("probe %s: %w", n.SourceURI, err)
                return
            }

            n.MediaInfo = mediaInfo
            errCh <- nil
        }(node)
    }

    // Collect results
    for i := 0; i < len(inputNodes); i++ {
        if err := <-errCh; err != nil {
            return err
        }
    }

    return nil
}
```

---

## Phase 5: Propagate Metadata

Flow media information through the graph:

```go
func (p *DefaultPlanner) propagateMetadata(graph *Graph) error {
    // Process nodes in topological order
    order, err := graph.TopologicalSort()
    if err != nil {
        return err
    }

    for _, nodeID := range order {
        node := graph.GetNode(nodeID)

        if node.Type == "operation" {
            // Ask operator to compute output metadata based on inputs
            operator := p.operatorRegistry.Get(node.Operator)

            inputNodes := graph.GetPredecessors(nodeID)
            inputMediaInfo := make([]*schemas.MediaInfo, len(inputNodes))
            for i, inputNode := range inputNodes {
                inputMediaInfo[i] = inputNode.MediaInfo
            }

            outputMediaInfo, err := operator.ComputeOutputMetadata(
                node.Params,
                inputMediaInfo,
            )
            if err != nil {
                return fmt.Errorf("node %s: %w", nodeID, err)
            }

            node.MediaInfo = outputMediaInfo
        }
    }

    return nil
}
```

### Example Metadata Propagation

```
Input: video1 (1920x1080, 30fps, 5min)
  ↓
Operation: trim (start=10s, duration=1min)
  → Output metadata: 1920x1080, 30fps, 1min
  ↓
Operation: scale (width=1280, height=720)
  → Output metadata: 1280x720, 30fps, 1min
  ↓
Operation: loudnorm (target_lufs=-16)
  → Output metadata: 1280x720, 30fps, 1min, audio normalized
```

Each operator implements `ComputeOutputMetadata()` to calculate how it transforms the input.

---

## Phase 6: Estimate Resources

Estimate CPU time, memory, and disk usage:

```go
type ResourceEstimator interface {
    EstimateNode(node *schemas.PlanNode) (*schemas.NodeEstimates, error)
    EstimateTotal(graph *Graph) (*schemas.ResourceEstimates, error)
}

type DefaultEstimator struct {
    // Benchmarked constants
    encodeTimePerSecond  map[string]time.Duration // e.g., h264: 0.5s per 1s of video
    memoryPerResolution  map[string]int64         // e.g., 1080p: 200MB
}

func (e *DefaultEstimator) EstimateNode(node *schemas.PlanNode) (*schemas.NodeEstimates, error) {
    if node.Type != "operation" || node.MediaInfo == nil {
        return nil, nil
    }

    operator := node.Operator
    duration := node.MediaInfo.Duration

    // CPU time estimation
    var cpuTime time.Duration
    switch operator {
    case "trim", "concat":
        // Fast operations (copy or minimal processing)
        cpuTime = time.Duration(duration * 0.1) * time.Second

    case "scale", "crop":
        // Video processing (depends on resolution)
        cpuTime = time.Duration(duration * 0.5) * time.Second

    case "loudnorm":
        // Two-pass operation
        cpuTime = time.Duration(duration * 1.0) * time.Second

    case "export":
        // Full encode
        codec := node.Params["codec"].(string)
        multiplier := e.getEncodeMultiplier(codec)
        cpuTime = time.Duration(duration * multiplier) * time.Second

    default:
        // Conservative estimate
        cpuTime = time.Duration(duration * 1.0) * time.Second
    }

    // Memory estimation
    resolution := fmt.Sprintf("%dx%d",
        node.MediaInfo.VideoStreams[0].Width,
        node.MediaInfo.VideoStreams[0].Height,
    )
    memory := e.memoryPerResolution[resolution]
    if memory == 0 {
        memory = 200 * 1024 * 1024 // 200MB default
    }

    // Disk estimation (intermediate files)
    bitrate := node.MediaInfo.Bitrate
    diskUsage := int64(duration * float64(bitrate) / 8)

    return &schemas.NodeEstimates{
        CPUTime:     cpuTime,
        MemoryUsage: memory,
        DiskUsage:   diskUsage,
    }, nil
}
```

### Total Estimation

```go
func (e *DefaultEstimator) EstimateTotal(graph *Graph) (*schemas.ResourceEstimates, error) {
    var totalCPUTime time.Duration
    var peakMemory int64
    var totalDisk int64

    for _, node := range graph.Nodes {
        if node.Estimates != nil {
            totalCPUTime += node.Estimates.CPUTime
            if node.Estimates.MemoryUsage > peakMemory {
                peakMemory = node.Estimates.MemoryUsage
            }
            totalDisk += node.Estimates.DiskUsage
        }
    }

    // Add safety margin
    estimatedDuration := time.Duration(float64(totalCPUTime) * 1.2)

    return &schemas.ResourceEstimates{
        TotalCPUTime:      totalCPUTime,
        PeakMemory:        peakMemory,
        TotalDiskSpace:    totalDisk * 3, // Input + intermediate + output
        EstimatedDuration: estimatedDuration,
    }, nil
}
```

---

## Phase 7: Optimize Plan

Optimize the execution plan to reduce overhead:

```go
type PlanOptimizer interface {
    Optimize(plan *schemas.ProcessingPlan) error
}

type DefaultOptimizer struct{}
```

### Optimization 1: Merge Linear Chains

Merge consecutive operations into a single FFmpeg command:

```go
func (o *DefaultOptimizer) mergeLinearChains(graph *Graph) error {
    // Find linear chains: A → B → C where each has single input/output
    chains := o.findLinearChains(graph)

    for _, chain := range chains {
        if o.canMerge(chain) {
            // Merge into single FFmpeg command
            mergedNode := o.createMergedNode(chain)
            graph.ReplaceChainWithNode(chain, mergedNode)
        }
    }

    return nil
}

func (o *DefaultOptimizer) canMerge(chain []*schemas.PlanNode) bool {
    // Can merge if:
    // 1. All operations are filtergraph-compatible
    // 2. No operations require two-pass (e.g., loudnorm)
    // 3. No intermediate outputs needed

    for _, node := range chain {
        if node.Operator == "loudnorm" {
            return false // Two-pass, can't merge
        }
        if node.Operator == "export" {
            return false // Final output, can't merge
        }
    }

    return true
}
```

Example:
```
Before: input → trim → scale → crop → output
After:  input → [trim+scale+crop] → output

Single FFmpeg command:
ffmpeg -i input.mp4 \
  -filter_complex '[0:v]trim=...,scale=...,crop=...[v]' \
  -map '[v]' output.mp4
```

### Optimization 2: Identify Copy Operations

Detect when lossless copy can be used instead of re-encoding:

```go
func (o *DefaultOptimizer) identifyCopyOperations(graph *Graph) error {
    for _, node := range graph.Nodes {
        if node.Operator == "trim" {
            // Check if trim is on keyframe boundaries
            if o.isKeyframeAligned(node) {
                node.Params["codec"] = "copy"
                node.Estimates.CPUTime /= 100 // Much faster
            }
        }
    }
    return nil
}
```

### Optimization 3: Parallelize Independent Operations

Identify operations that can run in parallel:

```go
func (o *DefaultOptimizer) identifyParallelOperations(graph *Graph) ([][]string, error) {
    stages, err := graph.ComputeExecutionStages()
    if err != nil {
        return nil, err
    }

    // Mark nodes in same stage as parallelizable
    for i, stage := range stages {
        if len(stage) > 1 {
            // Multiple nodes can run in parallel
            for _, nodeID := range stage {
                node := graph.GetNode(nodeID)
                node.Metadata["parallel_stage"] = i
            }
        }
    }

    return stages, nil
}
```

---

## Error Handling

```go
type PlanError struct {
    Code    string
    Message string
    NodeID  string // Which node caused the error
    Cause   error
}

func (e *PlanError) Error() string {
    return fmt.Sprintf("plan error at node %s: %s", e.NodeID, e.Message)
}

// Common errors
var (
    ErrCyclicDependency = &PlanError{Code: "CYCLIC_DEPENDENCY"}
    ErrUnreachableNode  = &PlanError{Code: "UNREACHABLE_NODE"}
    ErrInvalidReference = &PlanError{Code: "INVALID_REFERENCE"}
    ErrMediaProbe       = &PlanError{Code: "MEDIA_PROBE_FAILED"}
)
```

---

## Testing Strategy

### Unit Tests

```go
func TestBuildDAG(t *testing.T) {
    spec := &schemas.JobSpec{
        Inputs: []schemas.Input{
            {ID: "video1", Source: "test.mp4"},
        },
        Operations: []schemas.Operation{
            {Op: "trim", Input: "video1", Output: "trimmed"},
        },
        Outputs: []schemas.Output{
            {ID: "trimmed", Destination: "out.mp4"},
        },
    }

    planner := NewPlanner()
    plan, err := planner.Plan(context.Background(), spec)

    assert.NoError(t, err)
    assert.Equal(t, 3, len(plan.Nodes)) // input + op + output
    assert.Equal(t, 2, len(plan.Edges)) // input→op, op→output
}

func TestDetectCycle(t *testing.T) {
    spec := &schemas.JobSpec{
        Operations: []schemas.Operation{
            {Op: "op1", Input: "op2_out", Output: "op1_out"},
            {Op: "op2", Input: "op1_out", Output: "op2_out"}, // Cycle!
        },
    }

    planner := NewPlanner()
    _, err := planner.Plan(context.Background(), spec)

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cycle")
}
```

### Integration Tests

Test with real media files:

```go
func TestPlanWithMediaProbe(t *testing.T) {
    spec := &schemas.JobSpec{
        Inputs: []schemas.Input{
            {ID: "video", Source: "testdata/sample.mp4"},
        },
        Operations: []schemas.Operation{
            {Op: "scale", Input: "video", Output: "scaled",
             Params: map[string]interface{}{"width": 1280, "height": 720}},
        },
        Outputs: []schemas.Output{
            {ID: "scaled", Destination: "/tmp/output.mp4"},
        },
    }

    planner := NewPlanner()
    plan, err := planner.Plan(context.Background(), spec)

    assert.NoError(t, err)

    // Verify media info was probed
    inputNode := plan.Nodes[0]
    assert.NotNil(t, inputNode.MediaInfo)
    assert.Greater(t, inputNode.MediaInfo.Duration, 0.0)

    // Verify metadata propagation
    scaleNode := plan.Nodes[1]
    assert.Equal(t, 1280, scaleNode.MediaInfo.VideoStreams[0].Width)
    assert.Equal(t, 720, scaleNode.MediaInfo.VideoStreams[0].Height)
}
```

---

## Performance Considerations

### Caching Media Probes

```go
type CachedProber struct {
    prober MediaProber
    cache  map[string]*schemas.MediaInfo
    mu     sync.RWMutex
}

func (p *CachedProber) Probe(ctx context.Context, uri string) (*schemas.MediaInfo, error) {
    p.mu.RLock()
    if info, ok := p.cache[uri]; ok {
        p.mu.RUnlock()
        return info, nil
    }
    p.mu.RUnlock()

    info, err := p.prober.Probe(ctx, uri)
    if err != nil {
        return nil, err
    }

    p.mu.Lock()
    p.cache[uri] = info
    p.mu.Unlock()

    return info, nil
}
```

### Lazy Evaluation

Don't probe inputs until actually needed:

```go
type LazyPlanner struct {
    // Only probe when generating FFmpeg commands
    // Not during initial planning
}
```

---

## Future Enhancements

1. **Cost-Based Optimization**: Choose execution plan with lowest cost (time, disk, memory)
2. **Multi-Output Optimization**: Share computation when multiple outputs need similar processing
3. **Incremental Planning**: Support modifying plans without rebuilding from scratch
4. **Plan Caching**: Cache plans for identical JobSpecs
5. **GPU Awareness**: Detect GPU availability and use hardware acceleration

---

## Summary

The Planner transforms a declarative JobSpec into an optimized, executable ProcessingPlan through:

1. **DAG Construction**: Build dependency graph from operations
2. **Validation**: Detect cycles, unreachable nodes, incompatibilities
3. **Ordering**: Topological sort and execution stage computation
4. **Metadata Flow**: Propagate media information through the graph
5. **Resource Estimation**: Predict CPU, memory, disk requirements
6. **Optimization**: Merge operations, identify copy-only paths, parallelize

The output ProcessingPlan contains all information needed for execution: node dependencies, media properties, resource estimates, and optimization hints.

---

**Status**: Ready for implementation
