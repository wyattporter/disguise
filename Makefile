# programs
GO ?= /usr/bin/go

# names
DISGUISE     ?= disguise
DISGUISE_EXE ?= $(DISGUISE).exe

COVERAGE_PROF ?= $(DISGUISE).coverage
COVERAGE_HTML ?= $(COVERAGE_PROF).html

.PHONY: all
all: $(DISGUISE_EXE)

.PHONY: test
test: $(COVERAGE_HTML)

$(COVERAGE_HTML): $(COVERAGE_PROF)
	$(GO) tool cover -html=$(COVERAGE_PROF) -o $(COVERAGE_HTML)

$(COVERAGE_PROF):
	$(GO) test -timeout 10s -v -cover -race -coverprofile=$(COVERAGE_PROF) ./...

.PHONY: clean
clean:
	$(RM) $(DISGUISE_EXE) $(COVERAGE_PROF) $(COVERAGE_HTML)

%.exe: %.go
	$(GO) build -ldflags="-s -w" -o $@
