import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const __dirname = dirname(fileURLToPath(import.meta.url));

test('SDK package manifest exposes build metadata', () => {
  const manifest = JSON.parse(readFileSync(resolve(__dirname, '../package.json'), 'utf8'));
  assert.equal(manifest.name, '@tigerswap/sdk');
  assert.equal(manifest.main, 'dist/index.js');
  assert.equal(manifest.types, 'dist/index.d.ts');
});
