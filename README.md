# CarDan

[![Tests](https://github.com/theraccoonbear/CarDan/actions/workflows/test.yml/badge.svg)](https://github.com/theraccoonbear/CarDan/actions/workflows/test.yml)


**CarDan** is a Go module for loading and resolving YAML documents with controlled handling of anchors and aliases.

CarDan operates by:

- Parsing YAML into a full syntax tree (`*yaml.Node`) using **gopkg.in/yaml.v3**.
- Indexing all anchors (`&id`) and aliases (`*id`) present in the document.
- Allowing field-specific resolution of aliases into scalar values under user control.
- Optionally decoding the fully-resolved YAML tree into Go types.

CarDan does **not**:

- Impose meaning on your YAML fields.
- Validate schemas.
- Interpret or special-case field names.
- Mutate YAML structure unless explicitly commanded.

---

## Project Scope

CarDan focuses on **structural transparency** and **explicit user control**:

- Parse YAML into a raw syntax tree.
- Preserve all anchors and aliases.
- Provide manual alias resolution, scoped to specific fields.
- Decode into Go types only when explicitly requested.

CarDan uses **only** `gopkg.in/yaml.v3`.  
No other YAML libraries are involved.

---

## Intended Audience

CarDan is intended for engineers who treat YAML as structured data, not mere configuration.  
Use CarDan if you need:

- Access to anchors and aliases at parse time.
- Controlled flattening of alias references.
- Precision decoding into Go structs after structural control is complete.

CarDan is not a general-purpose configuration loader.  
It is a tool for projects that require direct control over YAML structure.

---

## Example

```go
package main

import (
	"log"
	"strings"

	"github.com/your-org/cardan"
)

type Job struct {
	Name    string   `yaml:"name"`
	Parents []string `yaml:"parents"`
}

func main() {
	r := strings.NewReader(`
defaults: &defaults
  retries: 3
  delay: 5

job1:
  <<: *defaults
  name: "job1"
`)

	doc, err := cardan.Load(r)
	if err != nil {
		log.Fatal(err)
	}

	if err := doc.ResolveRefs("<<"); err != nil {
		log.Fatal(err)
	}

	var m map[string]Job
	if err := doc.Unmarshal(&m); err != nil {
		log.Fatal(err)
	}

	log.Printf("Parsed jobs: %+v", m)
}
```

---

## License

MIT
