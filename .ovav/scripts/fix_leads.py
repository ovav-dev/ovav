import yaml, glob

LEADS_DIR = '/home/braka/Systems/OVAV/ovav/agents/leads/'
PERM = {
    'edit': 'allow',
    'bash': {
        '*': 'allow',
        'gh auth login*': 'deny',
        'gh auth token*': 'deny',
        'gh pr merge*': 'deny',
        'gh release *': 'deny',
        'git push -f *': 'deny',
        'npm install *': 'deny',
        'pip install *': 'deny',
        'apt install *': 'deny',
        'sudo *': 'deny',
        'python3 tools/install/*': 'deny',
        'python3 tools/protocols/*': 'deny',
    },
    'external_directory': {
        '/home/braka/*': 'allow',
        '/home/braka/Labs/mimocode/data/memory/*': 'allow',
        '/home/braka/Systems/OVAV': 'allow',
        '/tmp/opencode/*': 'allow',
        '*': 'deny',
    }
}

SKIP = ['dante']
fixed = 0

for f in sorted(glob.glob(LEADS_DIR + '*.yaml')):
    with open(f) as fh:
        data = yaml.safe_load(fh)
    
    lid = data.get('id', '?')
    name = data.get('name', '?')
    
    if lid in SKIP:
        continue
    
    changed = False
    
    if not data.get('steps'):
        data['steps'] = 50
        changed = True
    
    if not data.get('description'):
        data['description'] = 'Lead de ' + str(data.get('display_name', data.get('area', '?')))
        changed = True
    
    if not data.get('permission'):
        data['permission'] = PERM
        changed = True
    
    if changed:
        with open(f, 'w') as fh:
            yaml.dump(data, fh, default_flow_style=False, allow_unicode=True, sort_keys=False)
        fixed += 1
        print('  FIXED:', name)

print('\nTotal fixed:', fixed)
