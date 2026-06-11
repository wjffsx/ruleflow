# IoT Gateway Example

Demonstrates a realistic IoT gateway scenario: load a rule chain from YAML configuration, register builtin nodes, and evaluate streaming data points.

## Run

```bash
go run main.go
```

## What it covers

- **YAML-driven pipeline**: Loads a complete rule chain (drop low-quality data → filter analog/digital → value range check → route) from an embedded YAML string.
- **Builtin node registration**: Uses `builtin.RegisterAll(r)` to register all builtin condition and action nodes.
- **MapDataContext**: Simple data point creation for evaluation.
- **Pipeline routing**: Chain declares `pipeline_type` and `inputs` for future router integration.

## Sample YAML chain

The example includes rules for:
1. Dropping data points with quality < 192
2. Filtering by device type (analog vs digital)
3. Checking value ranges against upper/lower limits
4. Routing matched data to configured targets

## Key takeaway

In production, you'd load the YAML from a file (or config center) rather than embedding it. The `config.DefaultLoader` handles file loading, parsing, validation, and Registry-backed instantiation in one call.
