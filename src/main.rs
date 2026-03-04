mod api;
mod cli;
mod commands;
mod config;
mod display;
mod models;
mod storage;

use anyhow::Result;
use clap::Parser;

#[tokio::main]
async fn main() -> Result<()> {
    let cli = cli::Cli::parse();

    match cli.command {
        cli::Command::Auth => commands::auth::run().await,
        cli::Command::Sync => commands::sync::run().await,
        cli::Command::Log { n } => commands::log::run(n),
        cli::Command::Status => commands::status::run(),
    }
}
