#!/usr/bin/env bun

import { Command } from "commander";
import pkg from "../package.json";
import { registerData } from "./commands/data.ts";
import { registerExport } from "./commands/export.ts";
import { registerLog } from "./commands/log.ts";
import { registerReport } from "./commands/report.ts";
import { registerSetup } from "./commands/setup.ts";
import { registerShow } from "./commands/show.ts";
import { registerStats } from "./commands/stats.ts";
import { registerStatus } from "./commands/status.ts";
import { registerSync } from "./commands/sync.ts";
import { registerTrend } from "./commands/trend.ts";

const program = new Command()
  .name("c2")
  .description("Concept2 Logbook CLI")
  .version(pkg.version, "-v, --version");

registerSetup(program);
registerSync(program);
registerLog(program);
registerStatus(program);
registerTrend(program);
registerExport(program);
registerReport(program);
registerData(program);
registerShow(program);
registerStats(program);

if (process.argv.length <= 2) {
  program.outputHelp();
  process.exit(0);
}

program.parseAsync(process.argv).catch((err: Error) => {
  console.error(`Error: ${err.message}`);
  process.exit(1);
});
