import { describe, expect, test } from "bun:test";
import { cn } from "./utils";

describe("cn", () => {
  test("merges class names", () => {
    expect(cn("a", "b")).toBe("a b");
  });

  test("tailwind-merge resolves conflicting utilities", () => {
    expect(cn("px-2 px-4")).toBe("px-4");
  });

  test("ignores falsy values", () => {
    expect(cn("a", false && "b", undefined, null)).toBe("a");
  });
});
