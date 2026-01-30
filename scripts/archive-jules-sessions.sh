#!/bin/bash
set -e

# Function to display usage
usage() {
    echo "Usage: $0"
    echo "Archives (deletes) local git branches starting with 'jules-' that have been merged into main."
    echo "Also suggests commands to delete corresponding remote branches."
    exit 1
}

if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    usage
fi

# Configuration
BASE_BRANCH="origin/main"
PATTERN="jules-*"

# Ensure we have the latest information
echo "Fetching latest changes..."
git fetch -p origin || echo "Warning: git fetch failed. Proceeding with known state."

# --- Local Branches ---
echo "Checking for local branches matching '$PATTERN' merged into $BASE_BRANCH..."

# Get list of local branches merged into the base branch
# We use grep to filter for the pattern because 'git branch --list' matching is sometimes limited with --merged
# Regex explanation: ^$PATTERN matches branches starting with 'jules-'
MERGED_LOCAL=$(git branch --merged "$BASE_BRANCH" --format "%(refname:short)" | grep "^$PATTERN" || true)

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

if [ -z "$MERGED_LOCAL" ]; then
    echo "No merged local Jules sessions found."
else
    for branch in $MERGED_LOCAL; do
        if [ "$branch" = "$CURRENT_BRANCH" ]; then
            echo "Skipping current branch: $branch"
            continue
        fi
        echo "Deleting local branch: $branch"
        # -d ensures it's fully merged. Since we checked with --merged, this should pass.
        git branch -d "$branch"
    done
fi

echo ""

# --- Remote Branches ---
echo "Checking for remote branches matching '$PATTERN' merged into $BASE_BRANCH..."

# Get list of remote branches merged into the base branch
MERGED_REMOTE=$(git branch -r --merged "$BASE_BRANCH" --format "%(refname:short)" | grep "origin/jules-" || true)

if [ -z "$MERGED_REMOTE" ]; then
    echo "No merged remote Jules sessions found."
else
    echo "Found the following merged remote branches:"
    echo "$MERGED_REMOTE"
    echo ""
    echo "To delete them, verify and run the following commands:"
    for branch in $MERGED_REMOTE; do
        # Strip 'origin/' prefix
        branch_name=${branch#origin/}
        echo "  git push origin --delete $branch_name"
    done
fi
