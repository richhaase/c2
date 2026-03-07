#!/usr/bin/env bun

import { Command } from "commander";
import { registerSetup } from "./commands/setup.ts";
import { registerSync } from "./commands/sync.ts";
import { registerLog } from "./commands/log.ts";
import { registerStatus } from "./commands/status.ts";
import { registerTrend } from "./commands/trend.ts";
import { registerExport } from "./commands/export.ts";
import { registerReport } from "./commands/report.ts";

const VERSION = "0.1.0";

const program = new Command()
  .name("c2")
  .description("Concept2 Logbook CLI")
  .version(VERSION, "-v, --version");

registerSetup(program);
registerSync(program);
registerLog(program);
registerStatus(program);
registerTrend(program);
registerExport(program);
registerReport(program);

program.parseAsync(process.argv).catch((err: Error) => {
  console.error(`Error: ${err.message}`);
  process.exit(1);
});
