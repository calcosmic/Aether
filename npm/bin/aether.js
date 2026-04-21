#!/usr/bin/env node

const { main } = require("../lib/bootstrap");

main(process.argv.slice(2)).catch((error) => {
  const message = error && error.message ? error.message : String(error);
  console.error(message);
  process.exit(1);
});
