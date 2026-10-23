"use strict";

const fill = require("fill-range");
const utils = require("./utils.cjs");
const guard = require("../../guard.cjs");

const compile = (ast, options = {}) => {
  guard.ast(ast);
  const walk = (node, parent = {}) => {
    const invalidBlock = utils.isInvalidBrace(parent);
    const invalidNode = node.invalid === true && options.escapeInvalid === true;
    const invalid = invalidBlock === true || invalidNode === true;
    const prefix = options.escapeInvalid === true ? "\\" : "";
    let output = "";

    if (node.isOpen === true) {
      return guard.output(prefix + node.value);
    }

    if (node.isClose === true) {
      console.log("node.isClose", prefix, node.value);
      return guard.output(prefix + node.value);
    }

    if (node.type === "open") {
      return guard.output(invalid ? prefix + node.value : "(");
    }

    if (node.type === "close") {
      return guard.output(invalid ? prefix + node.value : ")");
    }

    if (node.type === "comma") {
      return node.prev.type === "comma" ? "" : invalid ? node.value : "|";
    }

    if (node.value) {
      return guard.output(node.value);
    }

    if (node.nodes && node.ranges > 0) {
      const args = utils.reduce(node.nodes);
      guard.range(args, options, true);
      const range = fill(...args, {
        ...options,
        wrap: false,
        toRegex: true,
        strictZeros: true,
      });

      if (range.length !== 0) {
        return guard.output(
          args.length > 1 && range.length > 1 ? `(${range})` : range,
        );
      }
    }

    if (node.nodes) {
      for (const child of node.nodes) {
        const part = walk(child, node);
        if (output.length + String(part).length > guard.MAX_OUTPUT_BYTES)
          guard.refuse("compiled output length");
        output += part;
      }
    }

    return output;
  };

  return walk(ast);
};

module.exports = compile;
