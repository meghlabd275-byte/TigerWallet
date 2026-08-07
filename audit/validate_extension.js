const fs = require('fs');
const { execFileSync } = require('child_process');
const root = '/home/ubuntu/TigerWallet/browser_extension/chrome';
const manifest = JSON.parse(fs.readFileSync(`${root}/manifest.json`, 'utf8'));
const paths = [manifest.action.default_popup, manifest.background.service_worker, ...manifest.content_scripts.flatMap((entry) => entry.js)];
for (const relativePath of paths) {
  const absolutePath = `${root}/${relativePath}`;
  if (!fs.existsSync(absolutePath)) throw new Error(`missing ${relativePath}`);
}
const scripts = [
  `${root}/dist/service-worker.js`,
  `${root}/dist/inject.js`,
  ...fs.readdirSync(`${root}/dist`).filter((name) => name.endsWith('.js') && !['service-worker.js', 'inject.js'].includes(name)).map((name) => `${root}/dist/${name}`),
];
for (const script of scripts) execFileSync(process.execPath, ['--check', script], { stdio: 'inherit' });
console.log(`validated ${paths.length} manifest references and ${scripts.length} JavaScript files`);
