#!/usr/bin/env bash
# Validate the exact changelog commit before pushing it to protected main.
set -euo pipefail

: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
: "${GITHUB_RUN_ID:?GITHUB_RUN_ID is required}"
release_tag=${1:?release tag is required}
validation_branch=''

cleanup() {
  if [[ -n "$validation_branch" ]]; then
    git push origin --delete "$validation_branch" || echo "::warning::Could not delete $validation_branch"
  fi
}
trap cleanup EXIT

for attempt in 1 2 3; do
  validation_sha=$(git rev-parse HEAD)
  validation_branch="release/changelog-${release_tag}-${GITHUB_RUN_ID}-${attempt}"
  git push origin "HEAD:refs/heads/$validation_branch"

  # GITHUB_TOKEN pushes do not start workflows. Explicit dispatch does, and
  # creates real required checks on this SHA without granting a rules bypass.
  gh workflow run security-tests.yml --repo "$GITHUB_REPOSITORY" \
    --ref "$validation_branch" -f scan_published_image=false
  run_id=''
  for poll in {1..30}; do
    run_id=$(gh run list --repo "$GITHUB_REPOSITORY" \
      --workflow security-tests.yml --branch "$validation_branch" \
      --commit "$validation_sha" --event workflow_dispatch \
      --json databaseId --jq '.[0].databaseId // empty')
    [[ -n "$run_id" ]] && break
    sleep 2
  done
  if [[ -z "$run_id" ]]; then
    echo '::error::Release validation workflow did not start'
    exit 1
  fi
  timeout 25m gh run watch "$run_id" --repo "$GITHUB_REPOSITORY" --exit-status
  if [[ "$(git rev-parse HEAD)" != "$validation_sha" ]]; then
    echo '::error::Release commit changed during validation'
    exit 1
  fi
  if git push origin HEAD:refs/heads/main; then
    echo "Pushed validated release commit $validation_sha"
    exit 0
  fi
  cleanup
  validation_branch=''
  if [[ "$attempt" -eq 3 ]]; then
    echo '::error::Failed to push validated release after 3 attempts'
    exit 1
  fi
  # A concurrent merge changes the commit: rebase, then validate the new SHA.
  git pull --rebase origin main
done
