import yaml, glob

teams = sorted(glob.glob('/home/braka/Systems/OVAV/ovav/agents/teams/*.yaml'))
issues = []

for f in teams:
    data = yaml.safe_load(open(f))
    name = data.get('name', '?')
    lead = data.get('lead', '?')
    perm = data.get('permission', {})
    bash = perm.get('bash', {})
    ext_dir = perm.get('external_directory', {})
    
    # Check git push rules
    push_rules = [k for k in bash if 'push' in k.lower()]
    for rule in push_rules:
        val = bash[rule]
        if val not in ('deny', None):
            if 'force' in rule.lower():
                issues.append(f'CRITICAL: {name} force-push allowed: {rule}={val}')
    
    # Check dangerous commands
    dangerous = ['sudo *', 'npm install *', 'pip install *', 'apt install *']
    for d in dangerous:
        if d in bash and bash[d] != 'deny':
            issues.append(f'CRITICAL: {name}: {d}={bash[d]}')
    
    # Missing wildcard deny
    if bash and '*' not in bash:
        issues.append(f'MISSING: {name}: no wildcard deny in bash')
    
    # Cross-area external access
    if lead and lead != 'thavren' and ext_dir and len(ext_dir) > 1:
        allows = {k: v for k, v in ext_dir.items() if k != '*' and v == 'allow'}
        if allows:
            issues.append(f'CROSS-AREA: {name} ({lead}) allows: {list(allows.keys())}')

# MD agent audit
mds = sorted(glob.glob('/home/braka/Systems/OVAV/.mimocode/agents/team-*.md'))
md_issues = []
for f in mds:
    with open(f) as fh:
        c = fh.read()
    fm = c.split('---')[1] if c.count('---') >= 2 else ''
    n = f.split('/')[-1]
    if 'OVAV_IDENTITY_GUARD' not in c:
        md_issues.append(f'{n}: missing identity guard')
    if 'lead:' not in fm:
        md_issues.append(f'{n}: missing lead')
    # webfetch is inherited from runtime, not frontmatter — skip check

print(f'SECURITY: {len(issues)} issues')
for i in issues: print(f'  {i}')
print(f'MD: {len(md_issues)} issues')
for m in md_issues: print(f'  {m}')
print(f'SUMMARY: {len(issues)} sec, {len(md_issues)} md — {len(teams)} YAMLs, {len(mds)} MDs')
