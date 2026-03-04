use anyhow::Result;

use crate::config;
use crate::display;
use crate::storage;

pub fn run(n: usize) -> Result<()> {
    let cfg = config::load_config()?;
    let mut workouts = storage::read_workouts()?;

    if workouts.is_empty() {
        println!("No workouts found. Run `c2cli sync` first.");
        return Ok(());
    }

    // Sort by date descending
    workouts.sort_by(|a, b| b.date.cmp(&a.date));

    let show = workouts.iter().take(n);
    for w in show {
        println!("{}", display::format_workout_line(w, &cfg.display.date_format));
    }

    Ok(())
}
