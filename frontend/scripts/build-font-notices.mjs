import { mkdir, readFile, writeFile } from 'node:fs/promises';

const root = new URL('../', import.meta.url);
const pkg = JSON.parse(await readFile(new URL('package.json', root), 'utf8'));
const packages = Object.keys(pkg.devDependencies).filter((name) => name.startsWith('@fontsource/'));
const notices = ['Facet typography: third-party licences', ''];
for (const name of packages.sort()) {
	const directory = new URL('node_modules/' + name + '/', root);
	const installed = JSON.parse(await readFile(new URL('package.json', directory), 'utf8'));
	const licence = await readFile(new URL('LICENSE', directory), 'utf8');
	notices.push(name + '@' + installed.version, licence.trim(), '');
}
const target = new URL('static/fonts/', root);
await mkdir(target, { recursive: true });
await writeFile(new URL('THIRD-PARTY-NOTICES.txt', target), notices.join('\n'));
console.log('Bundled font licence notices for ' + packages.length + ' packages.');
