# Skill: Design System Component Generation

## Trigger
- Need UI components (button, card, form, modal)
- User mentions shadcn, design system, component library
- Request for consistent UI elements

## Prerequisites
- Design system registry loaded
- Framework selected (react/vue/svelte)

## Available Components

### Buttons
```json
{
  "button": {
    "variants": ["primary", "secondary", "ghost", "destructive", "outline"],
    "sizes": ["sm", "md", "lg", "icon"],
    "states": ["default", "hover", "active", "disabled", "loading"]
  }
}
```

**Usage:**
```
design_system_get_component(
  name="button",
  variant="primary",
  size="md",
  framework="react"
)
```

### Cards
```json
{
  "card": {
    "variants": ["default", "outlined", "elevated", "glass"],
    "slots": ["header", "content", "footer"]
  }
}
```

### Forms
```json
{
  "form": {
    "components": ["input", "select", "checkbox", "radio", "textarea", "switch"],
    "validation": ["required", "email", "min", "max", "pattern"]
  }
}
```

### Layout
```json
{
  "layout": {
    "components": ["container", "grid", "stack", "separator"],
    "responsive": true
  }
}
```

## Workflow

### 1. Get Component
```
design_system_get_component(
  name="button",
  variant="primary",
  size="md"
)
```

### 2. Get Token Value
```
design_system_get_token(
  category="colors",
  name="primary"
)
// Returns: hsl(var(--primary)) or #3B82F6
```

### 3. Validate Design
```
design_system_validate(
  file_path="src/components/Button.tsx"
)
// Returns: validation report with issues
```

## Component Examples

### Button
```tsx
// Generated with design system
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md text-sm font-medium",
  {
    variants: {
      variant: {
        primary: "bg-primary text-primary-foreground hover:bg-primary/90",
        secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline: "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
      },
      size: {
        sm: "h-9 px-3",
        md: "h-10 px-4 py-2",
        lg: "h-11 px-8",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "md",
    },
  }
)

interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  loading?: boolean
}

export function Button({ className, variant, size, loading, ...props }: ButtonProps) {
  return (
    <button
      className={cn(buttonVariants({ variant, size, className }))}
      disabled={loading}
      {...props}
    >
      {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
      {props.children}
    </button>
  )
}
```

### Card
```tsx
import { cn } from "@/lib/utils"

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: "default" | "outlined" | "elevated" | "glass"
}

export function Card({ className, variant = "default", ...props }: CardProps) {
  return (
    <div
      className={cn(
        "rounded-lg border bg-card text-card-foreground shadow-sm",
        {
          "border-transparent": variant === "glass",
          "shadow-md": variant === "elevated",
          "border-border": variant === "outlined",
        },
        className
      )}
      {...props}
    />
  )
}

export function CardHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex flex-col space-y-1.5 p-6", className)} {...props} />
}

export function CardContent({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("p-6 pt-0", className)} {...props} />
}

export function CardFooter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex items-center p-6 pt-0", className)} {...props} />
}
```

## Best Practices

### Token Usage
```tsx
// ✅ Good - Use tokens
<div className="p-4 gap-2">  // Uses spacing tokens
<Button className="bg-primary">  // Uses color token

// ❌ Bad - Hardcoded values
<div style={{ padding: '16px', gap: '8px'}}>
<Button style={{ backgroundColor: '#3B82F6'}}>
```

### Variant Composition
```tsx
// ✅ Good - Use variant system
<Button variant="primary" size="lg">

// ❌ Bad - Override styles manually
<Button className="bg-blue-500 text-white px-8 py-4">
```

## Validation Rules

| Rule | Severity | Description |
|------|----------|-------------|
| NO-HARDCODED-COLORS | error | Must use design tokens |
| NO-HARDCODED-SPACING | error | Must use spacing scale |
| ACCESSIBILITY-REVIEW | warning | Check aria labels |
| RESPONSIVE-CHECK | info | Verify mobile behavior |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Component not found" | Check component name in registry |
| "Invalid variant" | List variants with `design_system_list_variants` |
| "Token not found" | Verify token category and name |
