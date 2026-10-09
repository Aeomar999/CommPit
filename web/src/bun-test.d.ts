// Minimal ambient types for the bun:test runner (used by web/src unit
// tests). The real implementation is provided by bun at runtime; this file
// exists so tsc --noEmit passes without a @types/bun dependency.
declare module "bun:test" {
  export function describe(name: string, fn: () => void): void;
  export function test(name: string, fn: () => void | Promise<void>): void;
  export interface Matchers<T> {
    toBe(expected: T): void;
    toEqual(expected: T): void;
    toBeTruthy(): void;
  }
  export function expect<T>(actual: T): Matchers<T>;
}
