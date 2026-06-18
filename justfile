#!/usr/bin/env just --justfile

# This is justfile example for the standardisation of the CI/CD pipeline for GitLab.
# Use this justfile as a template and modify it according to your project's needs. CI/CD pipeline internally utilises recipes from this justfile structure.

# Load environment variables from `.env` file.
set dotenv-load

coverage_profile_log := "./deploy/coverage.out"
coverage_profile_html := "./deploy/coverage.html"
coverage_threshold := "90"

export BASE_PROJ_PATH := `pwd`

check: 
    # add your linting and formatting commands here
    echo "Running linting and formatting checks..."

install-deps:
    # add your dependency installation commands here
    echo "Installing dependencies..."

test:
    # add your test commands here for local testing (without coverage requirements)
    echo "Running tests..."    

test-with-coverage: 
    # IMPORTANT: This recipe must generate coverage artifacts for GitLab CI integration.
    #
    # ─────────────────────────────────────────────────────────────────────────────
    # REQUIRED OUTPUTS (all three must be satisfied):
    #
    # 1. coverage.out / .coverage  →  raw coverage data for diff-cover downstream job
    #    - Go:     {{ coverage_profile_log }}  (e.g. deploy/coverage.out)
    #    - Python: .coverage  (generated automatically by pytest-cov / coverage.py)
    #
    # 2. coverage.xml  →  Cobertura format, used for MR diff line annotations
    #
    # 3. Coverage % printed to stdout  →  feeds the GitLab MR widget and coverage history
    #    - Go:     must print a line matching:  ^total:\s+\(statements\)\s+(\d+\.\d+)%
    #    - Python: must print a line matching:  TOTAL.*? (100(?:\.0+)?%|[1-9]?\d(?:\.\d+)?%)
    # ─────────────────────────────────────────────────────────────────────────────
    #
    # CRITICAL: Always run from the repository root.
    #    Both gocover-cobertura (Go) and pytest-cov/coverage.py (Python) generate
    #    filename paths in coverage.xml relative to the working directory.
    #    GitLab maps these paths to actual repo files for MR diff annotations.
    #    Running from a subdirectory will produce wrong paths → no line annotations.
    #
    # ─────────────────────────────────────────────────────────────────────────────
    # EXAMPLE — Go:
    #
    #   go test ./... -coverprofile={{ coverage_profile_log }}
    #   go tool cover -func={{ coverage_profile_log }} # prints total line → MR widget
    #   gocover-cobertura < {{ coverage_profile_log }} > {{ coverage_profile_log.replace(".out", ".xml") }}  # generates coverage.xml → MR diff annotations
    #
    # EXAMPLE — Python (pytest-cov):
    #
    #   poetry run pytest \
    #       --cov=. \
    #       --cov-report=xml:coverage.xml \
    #       --cov-report=term-missing       # prints TOTAL line → MR widget
    #
    # ─────────────────────────────────────────────────────────────────────────────
    echo "Add your test-with-coverage commands here"
    
start *args='':
    # Start a local instance of a specific type of the service (server, consumer etc). if NO ARG present then it should bootstrap the entire ecosystem
    echo "Starting application..."

stop *args='':
    # Stop the specific server (if started). if no ARG then stop the entire ecosystem
    echo "Stopping application..."

integration-test:
    # add your integration testing commands here
    #
    echo "Running integration tests..."

migrate:
    # add your database migration commands here
    echo "Running database migrations..."