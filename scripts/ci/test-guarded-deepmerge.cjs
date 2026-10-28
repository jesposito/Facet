'use strict';
const assert = require('node:assert/strict');
const { createRequire } = require('node:module');
const path = require('node:path');
const fs = require('node:fs');
const crypto = require('node:crypto');
const root = path.resolve(__dirname, '../..');
const frontendRequire = createRequire(path.join(root, 'frontend/package.json'));
const merge = process.env.DEEPMERGE_TEST_MODULE ? require(process.env.DEEPMERGE_TEST_MODULE) : frontendRequire('deepmerge');
const poison = () => JSON.parse('{"__proto__":{"polluted":true},"constructor":{"prototype":{"polluted":true}},"prototype":{"polluted":true},"safe":1}');
const clean = (result) => {
  assert.equal(Object.getPrototypeOf(result), Object.prototype);
  assert.equal(result.polluted, undefined);
  for (const key of ['__proto__', 'constructor', 'prototype']) assert.equal(Object.hasOwn(result, key), false);
  assert.equal({}.polluted, undefined);
};
for (const target of [{}, Object.create(null)]) {
  const result = merge(target, poison());
  clean(result);
  assert.equal(result.safe, 1);
}
for (const target of [poison(), Object.assign(Object.create(null), poison())]) clean(merge(target, {}));
for (const value of [null, false, 0, '', undefined, {}]) {
  clean(merge({ child: value }, { child: poison() }).child);
  clean(merge({ child: poison() }, {}).child);
}
clean(merge([], [poison()])[0]);
let reads = 0;
const unread = {};
for (const key of ['__proto__', 'constructor', 'prototype']) Object.defineProperty(unread, key, { enumerable: true, get() { reads++; throw Error('forbidden getter'); } });
clean(merge(unread, {})); clean(merge({}, unread)); assert.equal(reads, 0);
const symbol = Symbol('safe');
const a = { nested: { left: 1 }, list: [{ id: 1 }], [symbol]: { a: 1 } };
const b = { nested: { right: 2 }, list: [{ id: 2 }], [symbol]: { b: 2 } };
const result = merge(a, b);
assert.deepEqual(result, { nested: { left: 1, right: 2 }, list: [{ id: 1 }, { id: 2 }], [symbol]: { a: 1, b: 2 } });
assert.notEqual(result.list[0], a.list[0]);
assert.deepEqual(a.nested, { left: 1 });
assert.equal(merge({}, a, { clone: false }).nested, a.nested);
assert.deepEqual(merge([1], [2], { arrayMerge: (_target, source) => source }), [2]);
assert.deepEqual(merge({ child: { a: 1 } }, { child: { b: 2 } }, { customMerge: key => key === 'child' ? () => ({ custom: true }) : undefined }), { child: { custom: true } });
assert.deepEqual(merge.all([{ a: 1 }, { b: 2 }]), { a: 1, b: 2 });
assert.throws(() => merge.all({}), /array/);
const date = new Date(0); assert.equal(merge({}, { date }).date, date);
assert.throws(() => merge(new Proxy({}, { has() { throw Error('blocked membership'); } }), { a: 1 }), /blocked membership/);
if (!process.env.DEEPMERGE_TEST_MODULE) {
  assert.equal(frontendRequire('deepmerge/package.json').name, '@facet/guarded-deepmerge');
  for (const dependency of ['@sveltejs/adapter-node', '@sveltejs/kit', 'svelte-i18n']) {
    const requireDependency = createRequire(frontendRequire.resolve(dependency === 'svelte-i18n' ? dependency : dependency + '/package.json'));
    assert.equal(requireDependency('deepmerge'), merge, dependency);
    clean(requireDependency('deepmerge')(Object.create(null), poison()));
  }
  const vendor = path.join(root, 'frontend/vendor/guarded-deepmerge');
  const provenance = JSON.parse(fs.readFileSync(path.join(vendor, 'PROVENANCE.json')));
  for (const [file, hash] of Object.entries(provenance.fork)) assert.equal(crypto.createHash('sha256').update(fs.readFileSync(path.join(vendor, file))).digest('hex'), hash);
}
console.log('guarded deepmerge: prototype guards, compatibility, transitive resolution and provenance passed');
