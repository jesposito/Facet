"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const { createRequire } = require("node:module");
const root = path.resolve(__dirname, "../..");
const supportedWorkspaces = ["frontend"];
const selectedWorkspaces = process.argv.slice(2);
if (
  selectedWorkspaces.some(
    (workspace) => !supportedWorkspaces.includes(workspace),
  )
) {
  throw new Error(
    "Expected zero or more workspace names: frontend",
  );
}
const workspaces = selectedWorkspaces.length
  ? selectedWorkspaces
  : supportedWorkspaces;
const limits = (error) =>
  error &&
  error.code === "ERR_FACET_BRACES_LIMIT" &&
  !(error instanceof RangeError);
let assertions = 0;
const same = (actual, expected) => {
  assert.deepEqual(actual, expected);
  assertions++;
};
const rejects = (call) => {
  assert.throws(call, limits);
  assertions++;
};

for (const workspace of workspaces) {
  const requireWorkspace = createRequire(
    path.join(root, workspace, "package.json"),
  );
  const braces = requireWorkspace("braces");
  const metadata = requireWorkspace("braces/package.json");
  same(metadata.name, "@facet/guarded-braces");
  same(metadata.version, "3.0.3-facet.1");
  same(metadata.license, "MIT");
  // These files really exist: a missing filename would not prove exports
  // close private bypasses. Bun reports MODULE_NOT_FOUND for export denial;
  // Node uses ERR_PACKAGE_PATH_NOT_EXPORTED. Neither may return an API.
  for (const privatePath of [
    "braces/upstream/lib/parse.cjs",
    "braces/upstream/lib/compile.cjs",
    "braces/upstream/lib/expand.cjs",
    "braces/upstream/lib/stringify.cjs",
    "braces/guard.cjs",
    "braces/index.cjs",
  ]) {
    assert.throws(
      () => requireWorkspace(privatePath),
      (error) =>
        ["ERR_PACKAGE_PATH_NOT_EXPORTED", "MODULE_NOT_FOUND"].includes(
          error.code,
        ),
    );
    assertions++;
  }
  for (const [pattern, compiled, expanded] of [
    ["{a,b}", "(a|b)", ["a", "b"]],
    ["{a,{b,c}}", "(a|(b|c))", ["a", "b", "c"]],
    ["{1..3}", "([1-3])", ["1", "2", "3"]],
    ["{03..01}", "(0[1-3])", ["03", "02", "01"]],
    [
      "./src/**/*.{html,js,svelte,ts}",
      "./src/**/*.(html|js|svelte|ts)",
      [
        "./src/**/*.html",
        "./src/**/*.js",
        "./src/**/*.svelte",
        "./src/**/*.ts",
      ],
    ],
    ["!(foo|bar)/{x,y}", "!(foo|bar)/(x|y)", ["!(foo|bar)/x", "!(foo|bar)/y"]],
    ["[{}]/{x,y}", "[{}]/(x|y)", ["[{}]/x", "[{}]/y"]],
  ]) {
    same(braces(pattern), [compiled]);
    same(braces.compile(braces.parse(pattern)), compiled);
    same(braces.stringify(braces.parse(pattern)), pattern);
    same(braces.expand(braces.parse(pattern)), expanded);
  }
  same(braces(["{x,y}", "{x,y}"], { expand: true, nodupes: true }), ["x", "y"]);
  same(braces.expand("{,a}", { noempty: true }), ["a"]);
  same(braces.expand("{1..3}", { rangeLimit: false }), ["1", "2", "3"]);
  same(
    braces.stringify(braces.parse("\\{x\\}", { keepEscaping: true })),
    "\\{x\\}",
  );
  same(braces('"' + "{".repeat(4000) + '"'), ["{".repeat(4000)]);
  same(braces("[" + "{".repeat(4000) + "]"), ["[" + "{".repeat(4000) + "]"]);
  same(braces("\\{".repeat(3500)), ["{".repeat(3500)]);
  assert.throws(() => braces.parse("abc", { maxLength: 2 }), SyntaxError);
  assertions++;
  assert.throws(() => braces.expand("{1..1001}"), RangeError);
  assertions++;
  const deep = "{".repeat(3500) + "a,b" + "}".repeat(3500);
  for (const method of [
    braces,
    braces.create,
    braces.parse,
    braces.compile,
    braces.expand,
    braces.stringify,
  ])
    rejects(() => method(deep));
  rejects(() => braces(["ordinary", deep]));
  rejects(() => braces("(".repeat(3500) + "x" + ")".repeat(3500)));
  rejects(() => braces(Array(1025).fill("{a,b}")));
  const deepAST = { type: "root", nodes: [] };
  let cursor = deepAST;
  for (let index = 0; index < 1000; index++) {
    const next = { type: "brace", nodes: [] };
    cursor.nodes.push(next);
    cursor = next;
  }
  const childCycle = { type: "root", nodes: [] };
  childCycle.nodes.push(childCycle);
  const parentCycle = { type: "paren", nodes: [] };
  parentCycle.parent = parentCycle;
  const largeAST = {
    type: "root",
    nodes: Array.from({ length: 20001 }, () => ({ type: "text", value: "x" })),
  };
  const largeValue = {
    type: "root",
    nodes: [{ type: "text", value: "x".repeat(1000001) }],
  };
  for (const method of [
    braces,
    braces.create,
    braces.compile,
    braces.expand,
    braces.stringify,
  ]) {
    for (const ast of [deepAST, childCycle, parentCycle, largeAST, largeValue])
      rejects(() => method(ast));
  }
  rejects(() => braces.expand("{1..20000}", { rangeLimit: false }));
  rejects(() => braces.expand("{20000..1}", { rangeLimit: false }));
  rejects(() => braces.expand("{a,b}".repeat(20)));
  // Moderate cases prove rejection before fill-range enumeration; never run
  // the billion-element advisory counterexample on a development machine.
  for (const method of [braces, braces.create, braces.compile]) {
    rejects(() => method("{1..40000..2}"));
    rejects(() => method(braces.parse("{40000..1..-2}")));
    rejects(() => method("{1..40000}", { step: "-2" }));
    rejects(() => method("{1..40000}", { step: {} }));
    rejects(() => method("{1..3}", { step: "0".repeat(10000) + "1" }));
    rejects(() =>
      method({
        type: "root",
        ranges: 1,
        nodes: ["1", "40000", "1", "extra"].map((value) => ({
          type: "text",
          value,
        })),
      }),
    );
  }
  same(braces.compile("{1..3}", { step: {} }), "(1,2,3)");
  same(braces.expand("{1..3..0}", { rangeLimit: false }), ["1", "2", "3"]);
  same(braces.expand("{1..3}", { step: 0, rangeLimit: false }), [
    "1",
    "2",
    "3",
  ]);
  same(braces.expand("{3..1}", { step: "0", rangeLimit: false }), [
    "3",
    "2",
    "1",
  ]);
  same(braces.compile("{1..5..-2}"), "(1|3|5)");
  same(braces.compile("{5..1..0}"), "([1-5])");
  const escapedLimit = {
    type: "open",
    isOpen: true,
    invalid: true,
    value: "x".repeat(1000000),
  };
  rejects(() => braces.compile(escapedLimit, { escapeInvalid: true }));
  rejects(() => braces.stringify(escapedLimit, { escapeInvalid: true }));
  // Each admitted range is small; their combined compiled regex is not.
  rejects(() => braces.compile("{1..10000..2}".repeat(50)));
  // Inspect the actual internal compile closure with its ordinary dependency
  // cache temporarily instrumented. Oversized cases must never reach fill.
  const compilePath = path.join(
    path.dirname(requireWorkspace.resolve("braces")),
    "upstream/lib/compile.cjs",
  );
  const fillPath = createRequire(compilePath).resolve("fill-range");
  const fillModule = require.cache[fillPath];
  const oldFill = fillModule.exports;
  const oldCompile = require.cache[compilePath];
  let fillCalls = 0;
  try {
    fillModule.exports = (...args) => {
      fillCalls++;
      return oldFill(...args);
    };
    delete require.cache[compilePath];
    const instrumented = require(compilePath);
    rejects(() => instrumented(braces.parse("{1..40000..2}")));
    rejects(() => instrumented(braces.parse("{40000..1}"), { step: "-2" }));
    rejects(() => instrumented(braces.parse("{1..40000}"), { step: {} }));
    rejects(() =>
      instrumented(braces.parse("{1..3}"), {
        step: "0".repeat(10000) + "1",
      }),
    );
    same(fillCalls, 0);
    same(instrumented(braces.parse("{1..5..2}")), "(1|3|5)");
    same(fillCalls, 1);
  } finally {
    fillModule.exports = oldFill;
    require.cache[compilePath] = oldCompile;
  }
  // Confirm transitive consumers use the real patched closure, not just the
  // direct source import. Both Tailwind entry and its nested watcher are checked.
  for (const dependency of [
    "micromatch",
    "tailwindcss",
    "tailwindcss/node_modules/chokidar",
  ]) {
    const dependencyRequire = createRequire(
      requireWorkspace.resolve(dependency),
    );
    const transitive = dependencyRequire("braces");
    same(transitive("{a,b}"), ["(a|b)"]);
    rejects(() => transitive(deep));
  }
  const match = requireWorkspace("micromatch");
  same(match(["a.js", "b.ts", "c.css"], "*.{js,ts}"), ["a.js", "b.ts"]);
  const fastGlob = requireWorkspace("fast-glob");
  const files = fastGlob.sync("./src/**/*.{html,js,svelte,ts}", {
    cwd: path.join(root, workspace),
  });
  assert(files.length > 0);
  assertions++;
  console.log(
    `guarded braces ${workspace}: actual dependency/AST/array/glob checks passed (${files.length} configured source files)`,
  );
}

// Check that the carried fork matches its recorded mitigation hashes.
const vendor = path.join(root, "frontend/vendor/guarded-braces");
const provenance = JSON.parse(fs.readFileSync(path.join(vendor, "PROVENANCE.json"), "utf8"));
for (const [filename, expected] of Object.entries(provenance.fork.mitigationFiles)) {
  same(crypto.createHash("sha256").update(fs.readFileSync(path.join(vendor, filename))).digest("hex"), expected);
}
console.log(`guarded braces: ${assertions} assertions passed; provenance verified`);
