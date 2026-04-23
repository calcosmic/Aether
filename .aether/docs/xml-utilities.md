# XML Utilities

## Status

Active and supported.

These utilities back XML export/import/validation flows used by runtime subcommands and by user-facing commands such as `/ant-seal`, `/ant-entomb`, and `/ant-tunnels`.

## Utility Files

- `pkg/exchange/`
- `pkg/exchange/`
- `pkg/exchange/`
- `pkg/exchange/`
- `pkg/exchange/`

## Runtime Integration

Core XML subcommands in the Go `aether` runtime:

- `pheromone-export-xml`
- `pheromone-import-xml`
- `pheromone-validate-xml`
- `wisdom-export-xml`
- `wisdom-import-xml`
- `registry-export-xml`
- `registry-import-xml`
- `colony-archive-xml`

These subcommands source exchange modules:

- `pkg/exchange/pheromone_xml.go`
- `pkg/exchange/wisdom_xml.go`
- `pkg/exchange/registry_xml.go`

## Schema Dependencies

- `.aether/schemas/pheromone.xsd`
- `.aether/schemas/queen-wisdom.xsd`
- `.aether/schemas/colony-registry.xsd`
- `.aether/schemas/aether-types.xsd`

## Operational Notes

- `xmllint` is required for validation/import/export flows that parse XML.
- XML features degrade gracefully when dependencies are missing, returning structured JSON errors.
- These utilities are included in `bootstrap-system` allowlist for recovery/update flows.
