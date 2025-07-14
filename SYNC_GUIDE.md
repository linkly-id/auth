# Linkly Auth - Upstream Sync Guide

## 🎯 Fork Maintenance Strategy

Your Linkly auth project is a **comprehensive rebrand** of Supabase auth with:
- Module name: `github.com/linkly-id/auth` 
- Extensive branding changes across ~200 files
- Custom test fixes and improvements

## 📋 Branch Structure

```
master                    ← Your production branch
├── linkly-customizations ← Your pure customizations  
├── upstream-sync         ← Clean upstream mirror
└── linkly-backup-*       ← Automatic backups
```

## 🔄 Regular Sync Workflow

### Option A: Automated (when stable)
```bash
./sync-upstream.sh
```

### Option B: Manual (recommended for major changes)
```bash
./manual-sync.sh
# Follow the interactive prompts
```

## ⚠️ Conflict Resolution

When you encounter conflicts during rebase:

### 1. Common Conflict Types
- **Import paths**: Always keep `github.com/linkly-id/auth`
- **Branding**: Keep "Linkly" over "Supabase"  
- **Go version**: Keep your `go 1.23.7` requirement
- **Test data**: Keep your linkly.xxx domains

### 2. Resolution Process
```bash
# When rebase stops on conflicts:
git status                    # See conflicted files
# Edit each file manually
git add .                     # Stage resolved files  
git rebase --continue         # Continue rebase
```

### 3. Smart Conflict Patterns
```bash
# In go.mod - ALWAYS keep:
module github.com/linkly-id/auth

# In any string - prefer:
"linkly" over "supabase"
"Linkly" over "Supabase"  

# In test files - keep:
linkly.xxx domains
Your custom signatures
```

## 🧪 Testing After Sync

```bash
# Essential tests after every sync:
make test                     # Run all tests
./go.sh version              # Check binary works
make build                   # Ensure compilation
```

## 📅 Recommended Sync Schedule

- **Weekly**: Check for new upstream releases
- **Monthly**: Full sync for minor updates  
- **Major releases**: Manual sync with careful testing

## 🚨 Emergency Rollback

```bash
# If sync breaks everything:
git checkout master
git reset --hard linkly-backup-YYYYMMDD-HHMMSS
git push origin master --force-with-lease
```

## 🎯 Key Customization Areas

Based on your changes, pay special attention to:
1. **Module paths** - Always linkly-id 
2. **Branding strings** - Linkly everywhere
3. **Test domains** - linkly.xxx for isolation
4. **Go version** - Maintain 1.23.7 compatibility
5. **Web3 signatures** - Your custom test data

## 📊 Monitoring Upstream

```bash
# Check what's new in upstream:
git log --oneline upstream/master ^master

# See version tags:
git tag -l 'v2.*' | tail -5
```
