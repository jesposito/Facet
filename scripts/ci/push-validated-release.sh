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
  base_sha=$(git rev-parse refs/remotes/origin/main)
  validation_sha=$(git rev-parse HEAD)
  validation_branch="release/changelog-${release_tag}-${GITHUB_RUN_ID}-${attempt}"
  git push origin "HEAD:refs/heads/$validation_branch"

  # GITHUB_TOKEN pushes do not start workflows, so explicitly run the tests.
  gh workflow run security-tests.yml --repo "$GITHUB_REPOSITORY" \
    --ref "$validation_branch"
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
  # GitHub does not credit dispatch check runs toward this protected push.
  # Report their actual results through the commit-status API, after checking
  # the exact SHA and both required jobs. Never report skipped/failed tests.
  run_info=$(gh run view "$run_id" --repo "$GITHUB_REPOSITORY" --json headSha,jobs)
  if ! jq -e --arg sha "$validation_sha" '
    .headSha == $sha and
    ([.jobs[] | select(.name == "Backend tests and security" or .name == "Frontend checks and smoke")]
      | sort_by(.name) | map({name, conclusion})) == [
        {"name": "Backend tests and security", "conclusion": "success"},
        {"name": "Frontend checks and smoke", "conclusion": "success"}
      ]
  ' <<< "$run_info" >/dev/null; then
    echo '::error::Release validation must pass both required jobs on the exact commit'
    exit 1
  fi
  for context in 'Backend tests and security' 'Frontend checks and smoke'; do
    gh api --method POST "repos/$GITHUB_REPOSITORY/statuses/$validation_sha" \
      -f state=success -f context="$context" \
      -f description='Verified successful job in release validation workflow' \
      -f target_url="https://github.com/$GITHUB_REPOSITORY/actions/runs/$run_id" --silent
  done
  if git push origin HEAD:refs/heads/main; then
    echo "Pushed validated release commit $validation_sha"
    exit 0
  fi
  cleanup
  validation_branch=''
  git fetch origin main
  if [[ "$(git rev-parse refs/remotes/origin/main)" == "$base_sha" ]]; then
    echo '::error::Main did not move; push rejection will not be fixed by rebuilding'
    exit 1
  fi
  if [[ "$attempt" -eq 3 ]]; then
    echo '::error::Failed to push validated release after 3 attempts'
    exit 1
  fi
  # A concurrent merge changes the commit: rebase, then validate the new SHA.
  git pull --rebase origin main
done
