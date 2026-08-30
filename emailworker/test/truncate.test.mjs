import { test } from "node:test";
import assert from "node:assert/strict";
import { truncateUtf8 } from "../src/truncate.js";

// A naive body.slice(0, N) caps by UTF-16 code units, not bytes. Vietnamese
// text runs 2-3 bytes per character in UTF-8, so that undercounts the real
// payload size by up to ~3x -- this fixture is long enough in bytes to catch
// that regression even though it is short in code units.
const vietnamese = "Ngân hàng Á Châu thông báo: tài khoản của quý khách vừa nhận được ".repeat(2000);

test("truncateUtf8 caps multi-byte text within the byte budget", () => {
  const maxBytes = 64 * 1024;
  const inputBytes = new TextEncoder().encode(vietnamese).length;
  assert.ok(inputBytes > maxBytes, "fixture must exceed the cap to be a real test");

  const got = truncateUtf8(vietnamese, maxBytes);
  const gotBytes = new TextEncoder().encode(got).length;

  assert.ok(gotBytes <= maxBytes, `truncated output is ${gotBytes} bytes, want <= ${maxBytes}`);
  assert.ok(!got.endsWith("�"), "truncated output must not end in a replacement character");

  // The cut is a real character boundary: re-encoding what we got must
  // round-trip to the same byte length, not silently drop a partial char.
  assert.equal(new TextEncoder().encode(got).length, gotBytes);
});

test("truncateUtf8 leaves a string under the cap untouched", () => {
  const short = "Ngân hàng Á Châu";
  assert.equal(truncateUtf8(short, 64 * 1024), short);
});
