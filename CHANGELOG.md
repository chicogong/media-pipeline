# Changelog

## [Unreleased] - 2025-12-16

### Added
- **Core Schemas**: Complete data structures for jobs, plans, and metadata
- **Operator Framework**: Extensible operator system with validation and type conversion
- **Built-in Operators**: trim and scale operators with FFmpeg compilation
- **Planner Module**: Complete DAG-based planning system
  - Graph construction with cycle detection
  - Topological sorting for execution order
  - Execution stage computation for parallelization
  - Metadata propagation through operations
  - Resource estimation (CPU, memory, disk)
- **Executor Module**: FFmpeg command execution engine
  - Command builder from processing plans
  - Real-time progress parsing
  - Process management with cancellation support
- **Documentation**: Comprehensive design documents and guides

### Project Structure
```
pkg/
├── schemas/      - Data structures (4 files, 400 lines)
├── operators/    - Operator framework (7 files, 800 lines)
├── planner/      - Planning system (13 files, 1,400 lines)
└── executor/     - Execution engine (7 files, 600 lines)

Total: 31 files, 3,200 lines of code + 1,900 lines of tests
```

### Progress
- ✅ Schemas: Complete
- ✅ Operators: Complete
- ✅ Planner: Complete
- ✅ Executor: Complete
- 📋 Store (Database): TODO
- 📋 API Server: TODO

**Overall**: ~60% complete
