#!/bin/bash

# Linkly Auth - Upstream Sync Script
# Maintains sync with Supabase auth while preserving Linkly customizations

set -e

echo "🔄 Starting upstream sync process..."

# 1. Fetch latest upstream changes
echo "📡 Fetching upstream changes..."
git fetch upstream

# 2. Update upstream-sync branch
echo "🔄 Updating upstream-sync branch..."
git checkout upstream-sync
git reset --hard upstream/master

# 3. Update linkly-customizations with latest changes
echo "🎯 Rebasing Linkly customizations..."
git checkout linkly-customizations
git rebase upstream-sync

if [ $? -ne 0 ]; then
    echo "⚠️  Rebase conflicts detected!"
    echo "📋 Resolve conflicts, then run:"
    echo "   git add ."
    echo "   git rebase --continue"
    echo "   ./sync-upstream.sh finish"
    exit 1
fi

# 4. Fast-forward master to match upstream
echo "⏩ Fast-forwarding master..."
git checkout master
git reset --hard upstream-sync

# 5. Apply Linkly customizations on top
echo "🔗 Applying Linkly customizations..."
git merge linkly-customizations --no-ff -m "feat: apply Linkly customizations on v$(git describe --tags upstream/master | sed 's/^v//')"

# 6. Push to origin
echo "📤 Pushing to origin..."
git push origin master --force-with-lease
git push origin linkly-customizations --force-with-lease
git push origin upstream-sync --force-with-lease

echo "✅ Upstream sync complete!"
echo "📊 Summary:"
echo "   - Master: $(git rev-parse --short HEAD)"
echo "   - Upstream: $(git rev-parse --short upstream/master)"
echo "   - Customizations: $(git rev-parse --short linkly-customizations)"
