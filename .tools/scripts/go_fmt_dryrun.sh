#!/bin/bash
# Check if Go files need formatting without modifying them

set -e

# Use gofmt -l to list files that need formatting
unformatted_files=$($(go env GOROOT)/bin/gofmt -l .)

if [ -n "$unformatted_files" ]; then
    echo "Files need formatting:"
    echo "$unformatted_files"
    echo "Run 'go fmt ./...' or 'just fmt' to fix formatting"
    exit 1
else
    echo "All files are properly formatted"
fi