import yaml, glob

RESPONSE_STYLE = {
    'max_words': 100,
    'format': 'result_first',
    'structure': 'icon + key_finding',
    'rules': [
        'Respuestas en español, ultra-compactas.',
        'Máximo 100 palabras por respuesta.',
        'Resultado primero, explicación después.',
        'Iconos (✅❌🔴🟢⚠️) cuando aplique.',
        'Cero frases de relleno.',
    ],
}

DOMAIN_HINTS = {
    'platform_engineering': 'Go runtime, validación, gobernanza técnica.',
    'digital_product': 'React, TypeScript, APIs, frontend/backend.',
    'ux_design': 'UX research, design systems, accesibilidad.',
    'research_intelligence': 'Investigación, evidencia, fuentes.',
    'commercial_growth': 'Estrategia comercial, pricing, growth.',
    'health_performance': 'Nutrición, fitness, bienestar.',
    'education_career': 'Educación, currículos, carrera.',
    'devops_infrastructure': 'CI/CD, cloud, SRE.',
    'adversarial_intelligence': 'Red team, pentesting, OWASP.',
    'legal_compliance': 'Contratos, compliance, regulación.',
}

teams_dir = '/home/braka/Systems/OVAV/ovav/agents/teams/'
fixed = 0
for f in sorted(glob.glob(teams_dir + '*.yaml')):
    with open(f) as fh:
        d = yaml.safe_load(fh)
    
    name = d.get('name', '?')
    area = d.get('area', '?')
    lid = d.get('id', '?')
    
    changed = False
    
    if not d.get('response_style'):
        d['response_style'] = RESPONSE_STYLE
        changed = True
    
    if not d.get('knowledge_rules'):
        domain = DOMAIN_HINTS.get(area, f'Especialista del área {area}.')
        d['knowledge_rules'] = {
            'domain': domain,
            'rules': [
                f'Especialista en {area}. Reporta a su lead.',
                'Conocer límites de la especialidad — escalar a lead o cross-area cuando aplique.',
                'HARD STOP fuera de la función: delegar al lead.',
            ],
        }
        changed = True
    
    if changed:
        with open(f, 'w') as fh:
            yaml.dump(d, fh, default_flow_style=False, allow_unicode=True, sort_keys=False)
        fixed += 1

print(f'Teams fixed: {fixed}/60')