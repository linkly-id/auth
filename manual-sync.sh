#!/bin/bash

# Linkly Auth - Manual First Sync
# Step-by-step sync for extensive customizations

echo "🎯 Linkly Auth - Manual Sync Process"
echo "===================================="

# Check current state
echo "📊 Current State:"
echo "  Master:    $(git rev-parse --short master)"
echo "  Upstream:  $(git rev-parse --short upstream/master)"
echo "  Linkly:    $(git rev-parse --short linkly-customizations)"
echo

# Create backup branch
echo "💾 Creating backup..."
git branch linkly-backup-$(date +%Y%m%d-%H%M%S) linkly-customizations

# Interactive rebase to clean up commits
echo "🧹 Ready to clean up your customizations..."
echo "   Your commits:"
git log --oneline upstream-sync..linkly-customizations

echo
echo "📋 Next Steps (run manually):"
echo "1. git checkout linkly-customizations"
echo "2. git rebase -i upstream-sync"
echo "   - Squash commits into logical chunks"
echo "   - Keep commit messages descriptive"
echo "3. Test the rebased code"
echo "4. git checkout master && git reset --hard linkly-customizations"
echo "5. git push origin master --force-with-lease"

echo
echo "🚨 IMPORTANT: Test your changes after each step!"
echo "   Run: make test"
echo "   Check: ./go.sh version"
