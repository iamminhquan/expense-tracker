import { test } from "node:test";
import assert from "node:assert/strict";
import { sign } from "../src/sign.js";

test("sign matches the Go side for a known fixture", async () => {
  const got = await sign("s3cret", "{}");
  assert.equal(got, "adbde1ce40c89c14215687d5d762a47df6dfaefcfad61e2e86718ffc8498571b");
});

test("sign is stable and secret-dependent", async () => {
  const a = await sign("k", "body");
  const b = await sign("k", "body");
  const c = await sign("other", "body");
  assert.equal(a, b);
  assert.notEqual(a, c);
});
