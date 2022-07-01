.POSIX:
.SUFFIXES:

MIGRATION_DIR = sqlite/migration

## help - display this help and exit
help: FORCE
	@grep '^##' Makefile \
		| sort -k2b \
		| awk -F' - ' 'BEGIN {print "Tasks:"} {sub(/## */, ""); printf("  %s\n        %s\n", $$1, $$2)}'

## create-migration - create a new SQL migration script
create-migration: FORCE
	@mkdir -p $(MIGRATION_DIR); \
	cd $(MIGRATION_DIR) || exit; \
	count=$$(find ./. -type f -name '*.sql' -print -o -name . -o -prune | wc -l); \
	file=$$(printf '%04d.sql' "$$count"); \
	touch "$$file"; \
	printf '$(MIGRATION_DIR)/%s\n' "$$file"

test: FORCE
	go test -race ./...

FORCE: ;
