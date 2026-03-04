use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "c2cli", about = "Concept2 Logbook CLI", version)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Command,
}

#[derive(Subcommand)]
pub enum Command {
    /// Authenticate with the Concept2 Logbook via OAuth2
    Auth,
    /// Sync workouts from the Concept2 Logbook API
    Sync,
    /// Show recent workouts
    Log {
        /// Number of workouts to display (default: 10)
        #[arg(default_value = "10")]
        n: usize,
    },
    /// Show progress toward the million-meter goal
    Status,
}
