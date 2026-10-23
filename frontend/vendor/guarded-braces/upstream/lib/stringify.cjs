"use strict";

const utils = require("./utils.cjs");
const guard = require("../../guard.cjs");

module.exports = (ast, options = {}) => {
  guard.ast(ast);
  const stringify = (node, parent = {}) => {
    const invalidBlock = options.escapeInvalid && utils.isInvalidBrace(parent);
    const invalidNode = node.invalid === true && options.escapeInvalid === true;
    let output = "";

    if (node.value) {
      if ((invalidBlock || invalidNode) && utils.isOpenOrClose(node)) {
        return guard.output("\\" + node.value);
      }
      return guard.output(node.value);
    }

    if (node.value) {
      return node.value;
    }

    if (node.nodes) {
      for (const child of node.nodes) {
        const part = stringify(child);
        if (output.length + String(part).length > guard.MAX_OUTPUT_BYTES)
          guard.refuse("stringified output length");
        output += part;
      }
    }
    return output;
  };

  return stringify(ast);
};
