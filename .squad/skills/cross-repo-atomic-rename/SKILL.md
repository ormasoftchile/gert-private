# SKILL: Cross-Repo Atomic Rename

**Slug:** `cross-repo-atomic-rename`  
**Author:** Tess (Conformance Tester)  
**Date:** 2026-06-07  
**Origin:** Schema rename `conformance/schema.json` → `vector.schema.json` (gert-private PR #8, gert PR #36)

---

## When to use

When a file is renamed in one repo and that same file (or path references to it) exists in one or more companion repos. The goal is two linked PRs that land together — neither repo is in a broken state independently, but reviewers see the companion link.

---

## Procedure

### 1. Identify blast radius before branching

```powershell
git grep -n "old-filename.ext"
git grep -n '"old-filename.ext"'   # catches relative refs inside the dir
```

Categorize hits into:
- **Active referrers** — code, living docs, spec files that will actually be used → update
- **Append-only logs** — decisions.md, history.md, archive → append migration note only, never rewrite
- **Historical records** — orchestration-log, archive, old decisions → leave untouched

### 2. Execute in the spec repo first

```powershell
git checkout -b tess/rename-<slug>
git mv old/path/file.ext old/path/new-file.ext
# Update all active referrers with edit tool
git add <only the rename files — NOT git add -A if other changes are staged>
git commit -m "rename: old-file.ext -> new-file.ext\n\nContext..."
git push origin tess/rename-<slug>
```

**Warning:** If other unstaged/pre-staged changes exist from prior sessions, `git add -A` will bundle them. Use explicit `git add <files>` to stay atomic.

### 3. Execute in the runtime repo

```powershell
cd P:\Projects\<runtime-repo>
git checkout -b tess/rename-<slug>
git mv old/path/file.ext new/path/new-file.ext
# Update all code/config referrers
git add <conformance dir or specific files>
git commit -m "chore(scope): track new-file.ext rename\n\nCompanion to..."
git push origin tess/rename-<slug>
```

### 4. Create PRs in the right order

Create **PR B (runtime)** first → get its URL.  
Create **PR A (spec)** → include PR B URL in body.  
Then `gh pr edit <B-number>` to add PR A URL to PR B's body.

This two-step avoids circular dependency on URL discovery.

```powershell
# PR B (runtime) first
cd P:\Projects\gert
gh pr create --title "chore(scope): track rename" --body "...Companion to PR A: to be linked..." --head tess/rename-<slug>
# Note returned URL, e.g. https://github.com/org/gert/pull/36

# PR A (spec)
cd P:\Projects\gert-private
gh pr create --title "rename: old -> new" --body "...Companion PR B: https://github.com/org/gert/pull/36..." --head tess/rename-<slug>
# Note returned URL, e.g. https://github.com/org/gert-private/pull/8

# Back-fill PR B with PR A URL
cd P:\Projects\gert
gh pr edit 36 --body "...PR A: https://github.com/org/gert-private/pull/8..."
```

### 5. Local validation (mandatory when CI is off)

**Schema validators:**  
Node.js Ajv (Draft 2020-12):
```powershell
npm install ajv ajv-formats js-yaml
node -e "
const Ajv=require('ajv/dist/2020'), addFormats=require('ajv-formats'), yaml=require('js-yaml'), fs=require('fs');
const schema=JSON.parse(fs.readFileSync('path/to/new-file.schema.json','utf8'));
const ajv=new Ajv({strict:false}); addFormats(ajv);
const v=ajv.compile(schema);
const files=fs.readdirSync('dir').filter(f=>f.startsWith('tv-')&&f.endsWith('.yaml'));
files.forEach(f=>{ const ok=v(yaml.load(fs.readFileSync('dir/'+f,'utf8'))); console.log(ok?'PASS':'FAIL',f); });
"
```

**Go tests:**
```powershell
$env:Path = "C:\Program Files\Go\bin;$env:Path"
go test ./internal/conformance/... -count=1
```

### 6. Append-only migration note format

For `decisions.md` and similar append-only logs:
```
Note: path renamed to vector.schema.json on YYYY-MM-DD per <decision-id>.
```

Do NOT edit historical lines. Do NOT edit archive files. Do NOT edit orchestration-log entries.

---

## Pitfalls

| Pitfall | Mitigation |
|---------|-----------|
| Pre-staged changes from other sessions get bundled | Always use explicit `git add <files>`, never `git add -A` on a shared branch |
| `node_modules/` appears in `git status` | Install validation deps ad hoc; don't commit `package.json` or `node_modules` |
| Schema `$id` may not match filename | Check `$id` field — update only if it contains the old filename |
| Circular PR-URL dependency | Create runtime PR first, then spec PR, then back-fill runtime PR body |
| History files contain old path strings | Leave them — they are records of what was done at that time, not live references |
