import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import { validateSession } from "../dist/index.js";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const schema = JSON.parse(readFileSync(join(root, "spec", "schema", "unified-transcript.schema.json"), "utf8"));
const ajv = new Ajv2020.default({ strict: false, allErrors: true });
addFormats.default(ajv);
const schemaValidate = ajv.compile(schema);

const fixtures = (kind) =>
  readdirSync(join(root, "spec", "fixtures", kind))
    .filter((name) => name.endsWith(".json"))
    .map((name) => [name, JSON.parse(readFileSync(join(root, "spec", "fixtures", kind, name), "utf8"))]);

for (const [name, session] of fixtures("valid")) {
  test(`valid/${name}: validator accepts`, () => {
    assert.deepEqual(validateSession(session), []);
  });
  test(`valid/${name}: schema accepts`, () => {
    assert.ok(schemaValidate(session), JSON.stringify(schemaValidate.errors));
  });
}

for (const [name, wrapper] of fixtures("invalid")) {
  test(`invalid/${name}: states a reason`, () => {
    assert.ok(typeof wrapper.reason === "string" && wrapper.reason.length > 0);
  });
  test(`invalid/${name}: validator rejects`, () => {
    assert.notDeepEqual(validateSession(wrapper.session), []);
  });
  test(`invalid/${name}: schema rejects`, () => {
    assert.equal(schemaValidate(wrapper.session), false);
  });
}
