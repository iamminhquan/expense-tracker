import { test } from "node:test";
import assert from "node:assert/strict";
import { sign } from "../src/sign.js";

// Fixture doi chieu voi Go: cung secret, cung body thi phai ra cung chuoi hex.
// internal/inbound/inbound_test.go asserts this exact same literal for
// inbound.Sign("s3cret", []byte("{}")) -- changing this constant on one side
// without changing the other silently breaks ingestion, because the Worker
// and the Go handler have to agree on every byte they sign.
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
