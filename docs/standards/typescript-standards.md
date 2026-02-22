# TypeScript Style Guide

<!-- Canonical source: gitops/docs/standards/typescript-standards.md -->
<!-- Last synced: 2026-01-19 -->

> **Purpose:** Standardized TypeScript conventions for type-safe application development.

## Scope

This document covers: strict configuration, ESLint integration, type system patterns, generic constraints, and utility types.

**Related:**
- [Python Style Guide](./python-style-guide.md) - Python coding conventions
- [Shell Script Standards](./shell-script-standards.md) - Bash scripting conventions

---

## Quick Reference

| Standard | Value | Validation |
|----------|-------|------------|
| **TypeScript** | 5.0+ | `tsc --version` |
| **Strict Mode** | Required | `"strict": true` in tsconfig.json |
| **Linter** | ESLint + typescript-eslint | `eslint . --ext .ts,.tsx` |
| **Formatter** | Prettier | `.prettierrc` at repo root |
| **Gate** | `tsc --noEmit` must pass | CI check |

---

## Strict Configuration

### tsconfig.json

Every TypeScript project MUST use strict mode:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "lib": ["ES2022"],
    "outDir": "./dist",
    "rootDir": "./src",

    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "exactOptionalPropertyTypes": true,

    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

**Why strict matters:**
- `strict: true` enables all strict type-checking options
- `noUncheckedIndexedAccess` adds `undefined` to index signatures
- `exactOptionalPropertyTypes` distinguishes `undefined` from missing

---

## ESLint Configuration

### eslint.config.js (Flat Config)

```javascript
import eslint from '@eslint/js';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  eslint.configs.recommended,
  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,
  {
    languageOptions: {
      parserOptions: {
        project: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/explicit-function-return-type': 'error',
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/prefer-nullish-coalescing': 'error',
      '@typescript-eslint/prefer-optional-chain': 'error',
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/await-thenable': 'error',
    },
  },
  {
    ignores: ['dist/', 'node_modules/', '*.js'],
  }
);
```

**Usage:**
```bash
# Lint check
npx eslint . --ext .ts,.tsx

# Fix auto-fixable issues
npx eslint . --ext .ts,.tsx --fix

# Type check only (no emit)
npx tsc --noEmit
```

---

## Type System Patterns

### Prefer Type Inference

Let TypeScript infer types when obvious:

```typescript
// Good - inference is clear
const users = ['alice', 'bob'];
const count = users.length;

// Good - explicit when non-obvious or API boundary
function getUser(id: string): User | undefined {
  return userMap.get(id);
}

// Bad - redundant annotation
const name: string = 'alice';
```

### Discriminated Unions

Use discriminated unions for state modeling:

```typescript
// Good - exhaustive pattern matching
type Result<T, E> =
  | { status: 'success'; data: T }
  | { status: 'error'; error: E };

function handleResult<T, E>(result: Result<T, E>): void {
  switch (result.status) {
    case 'success':
      console.log(result.data);
      break;
    case 'error':
      console.error(result.error);
      break;
    // TypeScript enforces exhaustiveness
  }
}
```

### Const Assertions

Use `as const` for literal types:

```typescript
// Good - preserves literal types
const CONFIG = {
  apiVersion: 'v1',
  retries: 3,
  endpoints: ['primary', 'fallback'],
} as const;

// Type: { readonly apiVersion: "v1"; readonly retries: 3; ... }
```

---

## Generic Constraints

### Constrained Generics

Always constrain generics when possible:

```typescript
// Good - constrained generic
function getProperty<T, K extends keyof T>(obj: T, key: K): T[K] {
  return obj[key];
}

// Good - multiple constraints
function merge<T extends object, U extends object>(a: T, b: U): T & U {
  return { ...a, ...b };
}

// Bad - unconstrained (allows any)
function unsafe<T>(value: T): T {
  return value;
}
```

### Generic Defaults

Provide defaults for optional type parameters:

```typescript
interface ApiResponse<T = unknown, E = Error> {
  data?: T;
  error?: E;
  status: number;
}

// Uses defaults
const response: ApiResponse = { status: 200 };

// Override defaults
const typed: ApiResponse<User, ApiError> = { status: 200 };
```

---

## Utility Types

### Built-in Utilities

Use built-in utility types over manual definitions:

```typescript
// Partial - all properties optional
type PartialUser = Partial<User>;

// Required - all properties required
type RequiredConfig = Required<Config>;

// Pick - select properties
type UserPreview = Pick<User, 'id' | 'name'>;

// Omit - exclude properties
type UserWithoutPassword = Omit<User, 'password'>;

// Record - typed object
type UserMap = Record<string, User>;

// Extract/Exclude - union manipulation
type StringOrNumber = Extract<string | number | boolean, string | number>;
```

### Custom Type Helpers

Create reusable type utilities:

```typescript
// Deep partial
type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

// Non-nullable object values
type NonNullableValues<T> = {
  [K in keyof T]: NonNullable<T[K]>;
};

// Extract function return types from object
type ReturnTypes<T extends Record<string, (...args: never[]) => unknown>> = {
  [K in keyof T]: ReturnType<T[K]>;
};
```

---

## Conditional Types

### Type-Level Logic

Use conditional types for dynamic typing:

```typescript
// Infer array element type
type ElementOf<T> = T extends readonly (infer E)[] ? E : never;

// Flatten promise type
type Awaited<T> = T extends Promise<infer U> ? Awaited<U> : T;

// Function parameter extraction
type FirstParam<T> = T extends (first: infer P, ...args: never[]) => unknown
  ? P
  : never;
```

### Template Literal Types

Use template literals for string manipulation:

```typescript
// Event handler naming
type EventName = 'click' | 'change' | 'submit';
type HandlerName = `on${Capitalize<EventName>}`;
// Result: "onClick" | "onChange" | "onSubmit"

// Path building
type ApiPath<T extends string> = `/api/v1/${T}`;
type UserPath = ApiPath<'users'>; // "/api/v1/users"
```

---

## Error Handling

### Result Pattern

Prefer explicit error handling over exceptions:

```typescript
type Result<T, E = Error> =
  | { ok: true; value: T }
  | { ok: false; error: E };

function parseJson<T>(json: string): Result<T, SyntaxError> {
  try {
    return { ok: true, value: JSON.parse(json) as T };
  } catch (e) {
    return { ok: false, error: e as SyntaxError };
  }
}

// Usage
const result = parseJson<User>(input);
if (result.ok) {
  console.log(result.value.name);
} else {
  console.error(result.error.message);
}
```

### Type Guards

Use type guards for runtime type narrowing:

```typescript
// User-defined type guard
function isUser(value: unknown): value is User {
  return (
    typeof value === 'object' &&
    value !== null &&
    'id' in value &&
    'name' in value
  );
}

// Assertion function
function assertUser(value: unknown): asserts value is User {
  if (!isUser(value)) {
    throw new Error('Invalid user');
  }
}
```

---

## Module Template

Standard template for TypeScript modules:

```typescript
/**
 * Module description.
 * @module module-name
 */

// Types first
export interface Config {
  readonly apiUrl: string;
  readonly timeout: number;
}

export type Handler<T> = (data: T) => Promise<void>;

// Constants
const DEFAULT_TIMEOUT = 5000;

// Private helpers (not exported)
function validateConfig(config: Config): void {
  if (!config.apiUrl) {
    throw new Error('apiUrl is required');
  }
}

// Public API
export function createClient(config: Config): Client {
  validateConfig(config);
  return new Client(config);
}

export class Client {
  readonly #config: Config;

  constructor(config: Config) {
    this.#config = config;
  }

  async fetch<T>(path: string): Promise<T> {
    const response = await fetch(`${this.#config.apiUrl}${path}`);
    return response.json() as Promise<T>;
  }
}
```

---

## Summary Checklist

| Requirement | Check |
|-------------|-------|
| `"strict": true` in tsconfig.json | Required |
| `noUncheckedIndexedAccess` enabled | Required |
| ESLint with typescript-eslint | Required |
| `tsc --noEmit` passes | CI gate |
| No `any` types | Enforced via ESLint |
| Explicit return types on exports | Required |
| Discriminated unions for states | Preferred |
| Type guards for runtime checks | Preferred |
| Generic constraints | Required when using generics |
| Built-in utility types | Preferred over custom |

**Key Takeaways:**
1. TypeScript 5.0+ with strict mode required
2. ESLint with `strictTypeChecked` config
3. `tsc --noEmit` must pass before merge
4. Prefer type inference, annotate at API boundaries
5. Use discriminated unions for state modeling
6. Always constrain generic type parameters
7. Use built-in utility types over manual definitions
8. Prefer Result pattern over thrown exceptions

---

## Common Errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| `TS2322: Type X not assignable to Y` | Type mismatch | Check types, add assertion or fix data |
| `TS2345: Argument of type X` | Wrong function argument | Check function signature |
| `TS2531: Object is possibly null` | Null safety check missing | Add `if (x)` or use `x!` if certain |
| `TS2339: Property does not exist` | Missing type definition | Add to interface or use type guard |
| `TS7006: Parameter has implicit any` | Missing type annotation | Add explicit type annotation |
| `TS2554: Expected N arguments, got M` | Wrong argument count | Check function signature |
| `TS2769: No overload matches` | Wrong argument types | Check overload signatures |
| `TS6133: Variable declared but never used` | Unused variable | Remove or prefix with `_` |
| `ESLint: @typescript-eslint/no-floating-promises` | Unhandled promise | Add `await` or `void` prefix |

---

## Anti-Patterns

| Name | Pattern | Why Bad | Instead |
|------|---------|---------|---------|
| Any Escape | `as any` or `as unknown as T` | Defeats type safety | Fix the types, use type guards |
| Non-null Assertion Spam | `x!.y!.z!` | Runtime errors if wrong | Proper null checks |
| Type-Only Imports Missing | `import { Type }` | Bundler includes unused | `import type { Type }` |
| Index Signature Abuse | `[key: string]: any` | No type safety | Explicit properties or generics |
| Enum for Strings | `enum Color { Red = "RED" }` | Verbose, poor tree-shaking | Union: `type Color = "RED" \| "BLUE"` |
| Interface for Everything | `interface X {}` for simple objects | Unnecessary abstraction | `type X = {...}` for simple cases |
| Callback Hell | Nested `.then()` chains | Hard to read/debug | async/await |

---

## AI Agent Guidelines

When AI agents write TypeScript for this repo:

| Guideline | Rationale |
|-----------|-----------|
| ALWAYS run `tsc --noEmit` before committing | Catches type errors |
| ALWAYS use `import type` for type-only imports | Smaller bundles |
| NEVER use `any` without comment explaining why | Maintain type safety |
| NEVER ignore ESLint errors with `// eslint-disable` | Fix the issue |
| PREFER `unknown` over `any` for untyped data | Safer, requires checks |
| PREFER `const` assertions for literals | Better type inference |
| PREFER discriminated unions for state | Exhaustiveness checking |
