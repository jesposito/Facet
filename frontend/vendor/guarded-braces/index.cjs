"use strict";

const upstream = require("./upstream/index.cjs");
const guard = require("./guard.cjs");

function check(value) {
  if (typeof value === "string") guard.input(value);
  else if (value && typeof value === "object") guard.ast(value);
}

const braces = (value, options = {}) => {
  if (Array.isArray(value)) {
    if (value.length > guard.MAX_PATTERNS) guard.refuse("pattern array length");
    for (const pattern of value) check(pattern);
  } else check(value);
  const patterns = Array.isArray(value) ? value : [value];
  let output = [];
  let bytes = 0;
  for (const pattern of patterns) {
    const values = [].concat(upstream.create(pattern, options));
    if (output.length + values.length > guard.MAX_RESULTS)
      guard.refuse("aggregate result count");
    for (const item of values) {
      if (typeof item === "string") bytes += item.length;
      if (bytes > guard.MAX_OUTPUT_BYTES)
        guard.refuse("aggregate output length");
      output.push(item);
    }
  }
  if (options && options.expand === true && options.nodupes === true)
    output = [...new Set(output)];
  return output;
};

for (const method of ["parse", "compile", "expand", "stringify", "create"]) {
  braces[method] = (value, options = {}) => {
    check(value);
    const result = upstream[method](value, options);
    if (method === "parse") guard.ast(result);
    return result;
  };
}

module.exports = braces;
