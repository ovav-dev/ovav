# Skill: Figma → Code Workflow

## Trigger
- User mentions Figma, design, mockup, prototype
- Request to convert design to code
- Need to extract design tokens

## Prerequisites
- Figma access token set in `FIGMA_TOKEN`
- File key from Figma URL

## Workflow

### 1. Get Design Information
```
figma_get_layout(
  fileKey="<file-key-from-url>",
  nodeId="<node-id>"
)
```

**Returns:**
- Layout structure (flex, grid, absolute)
- Component hierarchy
- Spacing and sizing values
- Responsive breakpoints

### 2. Extract Design Tokens
```
figma_get_tokens(
  fileKey="<file-key>",
  tokenType="all"  // or: colors, spacing, typography
)
```

**Returns:**
- Color palette (hex, RGB, HSL)
- Spacing scale
- Typography scale
- Border radius values

### 3. Generate Component
```
figma_generate_component(
  fileKey="<file-key>",
  nodeId="<node-id>",
  framework="react",  // or: vue, svelte
  designSystem="shadcn"  // or: custom
)
```

**Returns:**
- Component file (.tsx, .vue, .svelte)
- Styles (CSS modules, Tailwind, styled-components)
- Props interface
- Usage example

## Best Practices

### Do
- ✅ Always extract tokens first for consistency
- ✅ Use design system components when available
- ✅ Verify generated code against design system tokens
- ✅ Test responsive behavior

### Don't
- ❌ Skip token extraction (leads to inconsistent styles)
- ❌ Use hardcoded values instead of tokens
- ❌ Ignore accessibility attributes from Figma
- ❌ Generate components without proper spacing

## Example: Login Form

```
User: "Generate a login form from this Figma design"

Agent:
1. figma_get_layout(fileKey="abc123", nodeId="1:234")
   → Layout: Column, gap: 16px, padding: 24px

2. figma_get_tokens(fileKey="abc123", tokenType="all")
   → Colors: primary=#3B82F6, background=#FFFFFF
   → Spacing: sm=8px, md=16px, lg=24px
   → Typography: h2=24px, body=14px

3. figma_generate_component(
     fileKey="abc123",
     nodeId="1:234",
     framework="react",
     designSystem="shadcn"
   )

Output:
// Login.tsx
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card } from "@/components/ui/card"

export function Login() {
  return (
    <Card className="p-6">
      <h2 className="text-2xl font-bold mb-4">Login</h2>
      <form className="flex flex-col gap-4">
        <Input type="email" placeholder="Email" />
        <Input type="password" placeholder="Password" />
        <Button type="submit">Sign In</Button>
      </form>
    </Card>
  )
}
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Invalid file key" | Check URL: figma.com/file/<FILE_KEY>/... |
| "Node not found" | Use Figma dev mode to get exact node ID |
| "Token extraction failed" | Verify FIGMA_TOKEN has read access |
| "Component generation error" | Check framework parameter is valid |

## Security Notes

- Never commit Figma tokens to version control
- Use environment variables for tokens
- Limit token scope to read-only when possible
- Rotate tokens periodically
