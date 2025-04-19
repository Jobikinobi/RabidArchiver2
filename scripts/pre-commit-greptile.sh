#!/bin/bash

# Check if greptile is installed
if ! command -v greptile &> /dev/null; then
    echo "Error: greptile is not installed. Please install it with: go install github.com/greptile/greptile/cmd/greptile@latest"
    exit 1
fi

# Run greptile lint on all files
echo "Running Greptile lint..."
RESULT=$(greptile lint ./... --fix 2>&1)
EXIT_CODE=$?

# Display the result
echo "$RESULT"

# If there are still blockers, prevent the commit
if [ $EXIT_CODE -ne 0 ]; then
    echo "Error: Greptile found blocking issues. Please fix them before committing."
    exit 1
fi

echo "Greptile lint passed successfully."
exit 0 