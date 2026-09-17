#!/usr/bin/env bash
# One-shot: push the pending commits, deploy the CDK stack (OIDC role for
# GitHub Actions), wire the repo variables, and kick off the first deploy.
#
# Run from your own terminal (not the sandbox):   ./tmp-finish-deploy.sh
# Safe to re-run; every step is idempotent.
#
# Optional: SANDBOX=<name> ./tmp-finish-deploy.sh   also gives the Claude
# sandbox a GitHub token so it can push in future sessions.

set -euo pipefail
cd "$(dirname "$0")"

STACK=CodyOlsenRoot
REGION=us-east-1
REPO=codyolsen/codyolsen.com

step() { printf '\n\033[1;36m==> %s\033[0m\n' "$*"; }

step "Preflight"
gh auth status >/dev/null || { echo "gh is not logged in — run: gh auth login"; exit 1; }
aws sts get-caller-identity --query Account --output text >/dev/null \
  || { echo "AWS credentials not available — configure them first"; exit 1; }

step "Push main to $REPO"
git push -u origin main

if [[ -n "${SANDBOX:-}" ]]; then
  step "Give sandbox '$SANDBOX' a GitHub token (for future pushes from Claude)"
  sbx secret set "$SANDBOX" github -t "$(gh auth token)"
fi

step "Deploy CDK stack (adds GitHub OIDC provider + deploy role)"
(cd infra && cdk deploy "$STACK" --require-approval never)
# If this fails with "provider already exists" for
# token.actions.githubusercontent.com, the account already has a GitHub OIDC
# provider from another project — tell Claude and the stack will be switched
# to import it instead of creating one.

step "Read stack outputs"
out() {
  aws cloudformation describe-stacks --stack-name "$STACK" --region "$REGION" \
    --query "Stacks[0].Outputs[?OutputKey=='$1'].OutputValue" --output text
}
ROLE_ARN=$(out GitHubDeployRoleArn)
BUCKET=$(out SiteBucketName)
DIST_ID=$(out DistributionId)
echo "role:         $ROLE_ARN"
echo "bucket:       $BUCKET"
echo "distribution: $DIST_ID"
[[ -n "$ROLE_ARN" && -n "$BUCKET" && -n "$DIST_ID" ]] || { echo "missing a stack output — aborting"; exit 1; }

step "Set CI token for the private Blowfish fork (repo secret GH_MODULES_TOKEN)"
if [[ -z "${BLOWFISH_TOKEN:-}" ]]; then
  echo "Best: a fine-grained PAT with Contents: Read on codyolsen/blowfish only"
  echo "  -> https://github.com/settings/personal-access-tokens/new"
  read -r -s -p "Paste token (or press Enter to reuse your gh CLI token): " BLOWFISH_TOKEN; echo
  BLOWFISH_TOKEN=${BLOWFISH_TOKEN:-$(gh auth token)}
fi
gh secret set GH_MODULES_TOKEN -R "$REPO" --body "$BLOWFISH_TOKEN"

step "Set GitHub repo variables"
gh variable set AWS_DEPLOY_ROLE_ARN -R "$REPO" --body "$ROLE_ARN"
gh variable set SITE_BUCKET -R "$REPO" --body "$BUCKET"
gh variable set CLOUDFRONT_DISTRIBUTION_ID -R "$REPO" --body "$DIST_ID"

step "Trigger and watch the deploy workflow"
gh workflow run deploy.yml -R "$REPO" --ref main
sleep 5
RUN_ID=$(gh run list -R "$REPO" --workflow=deploy.yml --limit 1 --json databaseId --jq '.[0].databaseId')
gh run watch "$RUN_ID" -R "$REPO" --exit-status

step "Done — site deployed. https://codyolsen.com"
echo "This script is disposable: rm tmp-finish-deploy.sh"
