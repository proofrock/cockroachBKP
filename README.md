# 🪳 cockroachdb-bkp

This is a simple Go(lang) application to dump the structure and data in a serverless instance of a CockroachDB
database as SQL statements.

# Build

```
make build
```

# Usage

Compile and run, with the only argument being the connection string CockroachLabs provides. E.g.:

```
cockroach_bkp "postgresql://<usr>:<pwd>@free-tier7.aws-eu-west-1.cockroachlabs.cloud:26257/<db>?sslmode=verify-full&sslrootcert=root.crt&options=--cluster%3D<cluster>"
```

Output can be redirected to a file, and restored via any SQL editor/manager.