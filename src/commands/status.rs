use anyhow::{Context, Result};
use chrono::{Datelike, Local};

use crate::config;
use crate::display;
use crate::storage;

pub fn run() -> Result<()> {
    let cfg = config::load_config()?;
    let workouts = storage::read_workouts()?;

    let target = cfg.goal.target_meters;
    let start = cfg
        .goal
        .start_date
        .context("goal.start_date not set in config")?;
    let end = cfg
        .goal
        .end_date
        .context("goal.end_date not set in config")?;
    let today = Local::now().date_naive();

    // Total meters in the goal period
    let total_meters: u64 = workouts
        .iter()
        .filter_map(|w| {
            let d = w.parsed_date()?.date();
            if d >= start && d <= end {
                Some(w.distance)
            } else {
                None
            }
        })
        .sum();

    let progress = total_meters as f64 / target as f64;
    let total_weeks = ((end - start).num_days() as f64 / 7.0).ceil() as u64;
    let weeks_elapsed = if today > start {
        ((today - start).num_days() as f64 / 7.0).floor() as u64
    } else {
        0
    };
    let remaining_meters = target.saturating_sub(total_meters);
    let remaining_weeks = total_weeks.saturating_sub(weeks_elapsed).max(1);
    let required_pace = remaining_meters / remaining_weeks;

    println!("Goal: {}m", display::format_meters(target));
    println!("Season start: {}", start.format("%Y-%m-%d"));
    println!(
        "Progress: {} / {} ({})",
        display::format_meters(total_meters),
        display::format_meters(target),
        display::format_percent(progress)
    );
    println!("Weeks elapsed: {} / {}", weeks_elapsed, total_weeks);
    println!(
        "Required pace: {}",
        display::format_meters_per_week(required_pace)
    );
    println!();

    // Last 4 weeks breakdown
    println!("Last 4 weeks:");
    for i in 0..4 {
        let week_end = today - chrono::Duration::days(i * 7);
        let week_start = week_end - chrono::Duration::days(6);
        // Align to Monday
        let week_start_aligned = week_start
            - chrono::Duration::days(week_start.weekday().num_days_from_monday() as i64);

        let (meters, sessions) = workouts
            .iter()
            .filter_map(|w| {
                let d = w.parsed_date()?.date();
                if d >= week_start_aligned && d < week_start_aligned + chrono::Duration::days(7) {
                    Some(w.distance)
                } else {
                    None
                }
            })
            .fold((0u64, 0usize), |(m, c), d| (m + d, c + 1));

        println!(
            "  Week of {}: {} ({} sessions)",
            week_start_aligned.format(&cfg.display.date_format),
            display::format_meters(meters),
            sessions
        );
    }
    println!();

    // Weekly average and on-pace check
    if weeks_elapsed > 0 {
        let avg = total_meters / weeks_elapsed;
        let on_pace = avg as f64 >= (target as f64 / total_weeks as f64);
        println!(
            "Current avg: {} — {}",
            display::format_meters_per_week(avg),
            if on_pace { "on pace ✓" } else { "behind pace ✗" }
        );
    }

    Ok(())
}
