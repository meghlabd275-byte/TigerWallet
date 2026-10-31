/**
 * Runtime check for the KYC document-URL policy in src/utils/safeUrl.tsx.
 *
 * The repository has no JavaScript test runner configured for this app, so the
 * check is self-contained. Run it from the app directory:
 *
 *   npx esbuild scripts/urlguard.test.tsx --bundle --platform=node \
 *     --format=esm --outfile=/tmp/urlguard.mjs && node /tmp/urlguard.mjs
 *
 * The helper reads window.location.origin, so it is stubbed before the import.
 */
(globalThis as any).window = { location: { origin: 'https://admin.tigerwallet.example' } };

const cases: Array<[string, string | null]> = [
  ['/api/kyc/doc/abc.pdf', '/api/kyc/doc/abc.pdf'],
  ['https://admin.tigerwallet.example/api/kyc/doc/abc.pdf', '/api/kyc/doc/abc.pdf'],
  ['javascript:alert(document.cookie)', null],
  ['JaVaScRiPt:alert(1)', null],
  ['data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==', null],
  ['vbscript:msgbox(1)', null],
  ['//evil.example/steal', null],
  ['https://evil.example/phish', null],
  ['http://127.0.0.1:9093/admin', null],
  ['file:///etc/passwd', null],
  ['', null],
];

async function main() {
  const { safeDocumentURL } = await import('../src/utils/safeUrl');
  let failed = 0;
  for (const [input, expected] of cases) {
    const got = safeDocumentURL(input);
    const ok = got === expected;
    if (!ok) failed++;
    console.log(`${ok ? 'PASS' : 'FAIL'} ${JSON.stringify(input)} -> ${JSON.stringify(got)} (expected ${JSON.stringify(expected)})`);
  }
  if (safeDocumentURL(undefined) !== null || safeDocumentURL(42) !== null || safeDocumentURL({}) !== null) {
    console.log('FAIL non-string inputs were accepted');
    failed++;
  }
  console.log(failed === 0 ? 'ALL PASS' : `${failed} FAILURES`);
  process.exit(failed === 0 ? 0 : 1);
}
main();
