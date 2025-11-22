# Linter Configuration Notes

## Current Status

The project uses golangci-lint with comprehensive linting rules. As of the CI/CD pipeline setup, all critical, high, and medium priority issues have been resolved.

## Deferred Optimizations

The following low-priority linter warnings are intentionally deferred for future optimization:

### S1016 - Type Conversion Style (1 instance)
- **Location**: `internal/tui/dashboard.go:578`
- **Issue**: Suggests using type conversion instead of struct literal
- **Impact**: Purely stylistic, no functional difference
- **Reason for deferral**: Negligible benefit, current code is clear and explicit

### fieldalignment - Struct Field Ordering (31 instances)
- **Locations**: Various structs across `internal/tui/`, `internal/templates/`, `internal/api/`, `internal/config/`
- **Issue**: Struct fields could be reordered to reduce memory padding
- **Potential savings**: 8-40 bytes per struct instance (varies by struct)
- **Impact**: Minor memory optimization
- **Reason for deferral**:
  - Requires careful refactoring to avoid introducing bugs
  - Performance impact is negligible for a CLI application
  - Memory savings are minimal in context of application size
  - Risk/benefit ratio favors deferring until performance profiling shows actual bottlenecks

## Future Work

If performance profiling indicates memory pressure or if contributing to the project:

1. **fieldalignment**: Can be addressed systematically by:
   ```bash
   # Install fieldalignment tool
   go install golang.org/x/tools/cmd/fieldalignment@latest

   # Fix specific files
   fieldalignment -fix internal/tui/dashboard.go
   ```

2. **S1016**: Simple one-line fix when touching related code

## Why These Are Acceptable

1. **No functional impact**: These are optimizations, not bugs
2. **Minimal performance impact**: CLI tools are not memory-constrained
3. **Code clarity**: Current code is readable and maintainable
4. **Testing overhead**: Changes would require additional testing for minimal gain

## Related Configuration

See `.golangci.yml` for current linter configuration. The linters reporting these issues are:
- `gosimple` (S1016)
- `govet` with `fieldalignment` check

These linters remain enabled to catch issues in new code.
