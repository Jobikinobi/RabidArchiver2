#!/bin/bash

# Usage: ./scripts/snapshot.sh <issue-number> <snapshot-number>
# Example: ./scripts/snapshot.sh 1 2  # Creates tag snapshot/1-2

if [ $# -ne 2 ]; then
    echo "Usage: $0 <issue-number> <snapshot-number>"
    echo "Example: $0 1 2  # Creates tag snapshot/1-2"
    exit 1
fi

ISSUE_NUMBER=$1
SNAPSHOT_NUMBER=$2
TAG_NAME="snapshot/${ISSUE_NUMBER}-${SNAPSHOT_NUMBER}"

# Create and push the tag
git tag -a "$TAG_NAME" -m "Snapshot $ISSUE_NUMBER-$SNAPSHOT_NUMBER"
echo "Created tag $TAG_NAME"
echo "To push this tag, run: git push origin $TAG_NAME" 