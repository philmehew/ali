#!/bin/bash
set -euo pipefail

REPO="philmehew/ali"
BRANCH="main"

# Check current status
echo "🔍 Current security status:"
STATUS=$(gh api "repos/$REPO" --jq '.security_and_analysis.secret_scanning_push_protection.status // "disabled"')
echo "Secret scanning push protection: $STATUS"

# Enable secret scanning and push protection
SEC_FILE=$(mktemp) || exit 1
trap 'rm -f "$SEC_FILE"' EXIT

cat > "$SEC_FILE" <<'EOF'
{
  "security_and_analysis": {
    "secret_scanning": {"status": "enabled"},
    "secret_scanning_push_protection": {"status": "enabled"}
  }
}
EOF

if [ "$STATUS" != "enabled" ]; then
  if gh api "repos/$REPO" --method PATCH --input "$SEC_FILE" > /dev/null 2>&1; then
    echo "✅ Secret scanning and push protection enabled"
  else
    echo "❌ Failed to enable secret scanning" >&2
    exit 1
  fi
else
  echo "✅ Secret scanning and push protection already enabled"
fi

# Branch protection
PROT_FILE=$(mktemp) || exit 1
trap 'rm -f "$PROT_FILE"' EXIT

cat > "$PROT_FILE" <<'EOF'
{
  "required_status_checks": null,
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "required_approving_review_count": 0,
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false
  },
  "restrictions": null,
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "required_conversation_resolution": true,
  "required_signatures": false
}
EOF

if gh api "repos/$REPO/branches/$BRANCH/protection" --method PUT --input "$PROT_FILE" > /dev/null 2>&1; then
  echo "✅ Branch protection optimized (PRs required, linear history, no force pushes)"
else
  echo "❌ Failed to set branch protection" >&2
  exit 1
fi

# Actions permissions — allow GitHub-owned actions + goreleaser
# Two API calls needed: one for the main permissions, one for the allowed list
ACTIONS_FILE=$(mktemp) || exit 1
trap 'rm -f "$ACTIONS_FILE"' EXIT

cat > "$ACTIONS_FILE" <<'EOF'
{
  "enabled": true,
  "allowed_actions": "selected"
}
EOF

ACTIONS_STATUS=$(gh api "repos/$REPO/actions/permissions" --jq '.allowed_actions // "unknown"')
if [ "$ACTIONS_STATUS" != "selected" ]; then
  if gh api "repos/$REPO/actions/permissions" --method PUT --input "$ACTIONS_FILE" > /dev/null 2>&1; then
    echo "✅ Actions permissions set to selected"
  else
    echo "❌ Failed to set actions permissions" >&2
    exit 1
  fi
else
  echo "✅ Actions permissions already set to selected"
fi

# Set which specific actions are allowed
SELECTED_FILE=$(mktemp) || exit 1
trap 'rm -f "$SELECTED_FILE"' EXIT

cat > "$SELECTED_FILE" <<'EOF'
{
  "github_owned_allowed": true,
  "verified_allowed": false,
  "patterns_allowed": ["goreleaser/*"]
}
EOF

SELECTED_STATUS=$(gh api "repos/$REPO/actions/permissions/selected-actions" --jq '.github_owned_allowed // false')
PATTERNS_STATUS=$(gh api "repos/$REPO/actions/permissions/selected-actions" --jq '.patterns_allowed // [] | length')
if [ "$SELECTED_STATUS" != "true" ] || [ "$PATTERNS_STATUS" = "0" ]; then
  if gh api "repos/$REPO/actions/permissions/selected-actions" --method PUT --input "$SELECTED_FILE" > /dev/null 2>&1; then
    echo "✅ Allowed actions configured (GitHub-owned + goreleaser/*)"
  else
    echo "❌ Failed to set allowed actions list" >&2
    exit 1
  fi
else
  echo "✅ Allowed actions list already configured"
fi

# Verify everything
echo ""
echo "🔍 Final verification:"
FINAL_SCAN=$(gh api "repos/$REPO" --jq '.security_and_analysis.secret_scanning.status // "unknown"')
FINAL_PUSH=$(gh api "repos/$REPO" --jq '.security_and_analysis.secret_scanning_push_protection.status // "unknown"')
FINAL_REVIEWS=$(gh api "repos/$REPO/branches/$BRANCH/protection" --jq '.required_pull_request_reviews.required_approving_review_count // "none"')
FINAL_ACTIONS=$(gh api "repos/$REPO/actions/permissions" --jq '.allowed_actions // "unknown"')
FINAL_GITHUB_OWNED=$(gh api "repos/$REPO/actions/permissions/selected-actions" --jq '.github_owned_allowed // false')
FINAL_PATTERNS=$(gh api "repos/$REPO/actions/permissions/selected-actions" --jq '.patterns_allowed // [] | join(", ")')

echo "  Secret scanning: $FINAL_SCAN"
echo "  Push protection: $FINAL_PUSH"
echo "  Reviews required: $FINAL_REVIEWS"
echo "  Actions allowed: $FINAL_ACTIONS (GitHub-owned: $FINAL_GITHUB_OWNED, Patterns: $FINAL_PATTERNS)"

OK=true
if [ "$FINAL_SCAN" != "enabled" ]; then echo "❌ Secret scanning not enabled" >&2; OK=false; fi
if [ "$FINAL_PUSH" != "enabled" ]; then echo "❌ Push protection not enabled" >&2; OK=false; fi
if [ "$FINAL_REVIEWS" != "0" ]; then echo "❌ Review count not set to 0" >&2; OK=false; fi
if [ "$FINAL_ACTIONS" != "selected" ]; then echo "❌ Actions permissions not set to selected" >&2; OK=false; fi
if [ "$FINAL_GITHUB_OWNED" != "true" ]; then echo "❌ GitHub-owned actions not allowed" >&2; OK=false; fi

if [ "$OK" = true ]; then
  echo "✅ Repo secured! Check https://github.com/philmehew/ali/security"
else
  echo "⚠️  Some settings did not apply correctly" >&2
  exit 1
fi
