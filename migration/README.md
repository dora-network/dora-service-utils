# Spanner database migration utility

Run PostgreSQL schema migrations against a Google Cloud Spanner database.

This utility is intended for use in integration tests, and to implement migration
commands as a CLI tool for each service, allowing the migrations to be managed
in the service itself rather than use a separate tool to migrate the database
when it is needed.

SQL migration files must be written in PostgreSQL syntax, and must be named
in sequential numeric order, e.g. `001.sql`, `002.sql`, `003.sql`, etc.
There is no strict validation on the numbering but it must be remembered that
the files will be ordered according to the file system order. Therefore it is
necessary to prefix leading zeroes to the numbers to ensure the correct order.

Due to PostgreSQL allowing complex SQL statements that have semi-colon terminators
within the statement, each statement should be written inside start and end markers
`-- +START` and `-- +END` respectively. This allows the utility to split the file
and extract each statement to apply.

The end marker `-- +END` is optional, and if not present, the utility will assume
the next start marker it encounters or the end of the file is an implicit end marker.

The markers are case-insensitive and can be written in any case.

Migrations can also be nested in subdirectories, however the numbering sequence should
be maintained across all directories. This is because the utility will check the version
associated with the migration file name and will apply the migrations only if the
migration version is greater than the current version of the database.
