# CUE Data Source

## Summary

Add a `util_cue` data source and `cuedecode` provider function that evaluate [CUE](https://cuelang.org/) configuration and expose the result as native Terraform dynamic types.

## Motivation

- CUE provides validation, defaults, and composition that HCL/JSON lack for complex config.
- Teams already maintaining CUE configs shouldn't need to duplicate them into Terraform variables.
- A data source keeps the provider read-only and side-effect free.

## Decisions

- Output format
  - Dynamic type — native field access without `jsondecode()`.
- CUE evaluation library
  - `cuelang.org/go` SDK directly.
- Multiple files / packages
  - `path` accepts a file or directory. When a directory, loads all `.cue` files as a package. Auto-detects `cue.mod/` for module/import resolution. Optional `package` attribute selects which package if a directory has multiple.
- Validation errors
  - Terraform diagnostics with CUE source positions via `errors.Details`.
- Expressions
  - Omitted — users structure their CUE to export what they need.
- Tags vs. input values
  - Tags only — `map(string)` matching CUE's `@tag()` mechanism.
- Unification
  - Data source: `input` attribute (list of CUE strings) unified with the loaded file.
  - Function: variadic string parameters unified in order.
- Function scope
  - Pure string handling only — no filesystem access, no package loading.

## Specification

### Data Source: `util_cue`

```hcl
data "util_cue" "config" {
  # File or directory path. Directories load as a CUE package.
  path = "${path.module}/config.cue"

  # Optional: select package when directory has multiple.
  package = "prod"

  # Optional: tags injected via @tag() (cue eval -t key=value).
  tags = {
    env = "prod"
  }

  # Optional: additional CUE expressions unified with the loaded value.
  input = [<<CUE
    port: >1024
  CUE
  ]
}

output "port" {
  value = data.util_cue.config.result.port
}
```

**Attributes:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | yes | File or directory to evaluate |
| `package` | string | no | Package name to select (directory loading only) |
| `tags` | map(string) | no | Tag injection via `@tag()` |
| `input` | list(string) | no | CUE expressions to unify with the loaded value |
| `result` | dynamic | computed | The evaluated CUE value |

### Function: `provider::util::cuedecode`

```hcl
# Simple decode
provider::util::cuedecode(file("${path.module}/config.cue"))

# With inline constraints (variadic)
provider::util::cuedecode(
  file("${path.module}/config.cue"),
  <<CUE
    port: >1024
    name: =~"^[a-z]+$"
  CUE
)
```

Takes one or more CUE strings. All are compiled and unified in order. The final value must be concrete.

### CUE Type Mapping

| CUE | Terraform |
|-----|-----------|
| string | string |
| bool | bool |
| int | number |
| float | number |
| null | null |
| struct | object |
| list | tuple |
