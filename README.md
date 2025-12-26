# go

orm/
├── db/
│   ├── db.go              # Open DB, config, pooling
│   ├── tx.go              # Transaction helpers
│   └── prepare.go         # Prepared statement cache (optional)
│
├── query/
│   ├── builder.go         # SELECT/INSERT/UPDATE/DELETE builder
│   ├── where.go           # Conditions (Eq, In, And, Or)
│   ├── order.go           # ORDER BY
│   └── limit.go           # LIMIT/OFFSET
│
├── model/
│   ├── model.go           # Model metadata
│   └── tags.go            # Parse `db` tags
│
├── scan/
│   ├── scanner.go         # Row → struct mapping
│   └── nulls.go           # sql.Null* helpers
│
├── repo/
│   ├── repository.go      # Generic repository[T]
│   └── errors.go          # NotFound, Conflict, etc.
│
├── gen/
│   ├── gen.go             # Codegen entrypoint
│   ├── schema.go          # DB schema introspection
│   └── templates/         # Go templates for codegen
│
├── migrate/
│   └── migrate.go         # Optional migration runner
│
├── internal/
│   ├── sqlutil/
│   │   ├── placeholders.go
│   │   └── escape.go
│   └── reflectutil/
│       └── cache.go       # Minimal reflect cache (if needed)
│
├── examples/
│   └── users/
│       ├── model.go
│       ├── queries.go
│       └── main.go
│
├── go.mod
└── README.md
