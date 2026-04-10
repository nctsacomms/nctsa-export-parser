#!/bin/sh
# Sorts PDFs from raw_data/ into team_data/, individual_data/, and unparsed/
cd "$(dirname "$0")" && exec go run ./cmd/sort
