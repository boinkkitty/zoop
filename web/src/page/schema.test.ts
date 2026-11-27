import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import { parseDoc, SCHEMA_VERSION } from "./schema";

// Shared with the Go validator tests (internal/pages), so the editor and the server agree.
const fixtures = join(__dirname, "../../../testdata/page-docs");

type Fixture = { notebook: string; doc: unknown; why?: string };

function load(kind: "valid" | "invalid"): [string, Fixture][] {
  const dir = join(fixtures, kind);
  return readdirSync(dir)
    .filter((f) => f.endsWith(".json"))
    .map((f) => [f, JSON.parse(readFileSync(join(dir, f), "utf8")) as Fixture]);
}

describe("Page document schema", () => {
  it.each(load("valid"))("accepts %s", (_, fx) => {
    expect(parseDoc(fx.doc)).not.toBeNull();
  });

  it.each(load("invalid"))("refuses %s", (_, fx) => {
    expect(parseDoc(fx.doc), fx.why).toBeNull();
  });

  it("is on the same schemaVersion as the server", () => {
    const { schemaVersion } = JSON.parse(
      readFileSync(join(fixtures, "schema-version.json"), "utf8"),
    );
    expect(SCHEMA_VERSION).toBe(schemaVersion);
  });
});
