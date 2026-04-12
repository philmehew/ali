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
    "required_approving_review_count": 1,
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
  echo "✅ Branch protection optimized (1 approval, linear history, no force pushes)"
else
  echo "❌ Failed to set branch protection" >&2
  exit 1
fi

# Verify everything
echo ""
echo "🔍 Final verification:"
FINAL_SCAN=$(gh api "repos/$REPO" --jq '.security_and_analysis.secret_scanning.status // "unknown"')
FINAL_PUSH=$(gh api "repos/$REPO" --jq '.security_and_analysis.secret_scanning_push_protection.status // "unknown"')
FINAL_REVIEWS=$(gh api "repos/$REPO/branches/$BRANCH/protection" --jq '.required_pull_request_reviews.required_approving_review_count // "none"')

echo "  Secret scanning: $FINAL_SCAN"
echo "  Push protection: $FINAL_PUSH"
echo "  Reviews required: $FINAL_REVIEWS"

OK=true
if [ "$FINAL_SCAN" != "enabled" ]; then echo "❌ Secret scanning not enabled" >&2; OK=false; fi
if [ "$FINAL_PUSH" != "enabled" ]; then echo "❌ Push protection not enabled" >&2; OK=false; fi
if [ "$FINAL_REVIEWS" != "1" ]; then echo "❌ Review count not set to 1" >&2; OK=false; fi

if [ "$OK" = true ]; then
  echo "✅ Repo secured! Check https://github.com/philmehew/ali/security"
else
  echo "⚠️  Some settings did not apply correctly" >&2
  exit 1
fi
