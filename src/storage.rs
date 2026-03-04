use anyhow::{Context, Result};
use std::collections::HashSet;
use std::fs::{self, OpenOptions};
use std::io::{BufRead, BufReader, Write};
use std::path::PathBuf;

use crate::config;
use crate::models::{StrokeData, Workout};

/// Path to the workouts JSONL file.
fn workouts_path() -> Result<PathBuf> {
    Ok(config::data_dir()?.join("workouts.jsonl"))
}

/// Path to a workout's stroke data file.
fn strokes_path(workout_id: u64) -> Result<PathBuf> {
    Ok(config::data_dir()?.join("strokes").join(format!("{}.jsonl", workout_id)))
}

/// Read all workouts from the JSONL file.
/// Returns an empty vec if the file doesn't exist.
pub fn read_workouts() -> Result<Vec<Workout>> {
    let path = workouts_path()?;
    if !path.exists() {
        return Ok(Vec::new());
    }

    let file = fs::File::open(&path)
        .with_context(|| format!("failed to open {}", path.display()))?;
    let reader = BufReader::new(file);
    let mut workouts = Vec::new();

    for (i, line) in reader.lines().enumerate() {
        let line = line.with_context(|| format!("failed to read line {}", i + 1))?;
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        let workout: Workout = serde_json::from_str(line)
            .with_context(|| format!("failed to parse workout on line {}", i + 1))?;
        workouts.push(workout);
    }

    Ok(workouts)
}

/// Append new workouts to the JSONL file, skipping any with IDs already present.
/// Returns the number of workouts actually written.
pub fn append_workouts(new_workouts: &[Workout]) -> Result<usize> {
    let existing = read_workouts()?;
    let existing_ids: HashSet<u64> = existing.iter().map(|w| w.id).collect();

    let path = workouts_path()?;
    let mut file = OpenOptions::new()
        .create(true)
        .append(true)
        .open(&path)
        .with_context(|| format!("failed to open {} for appending", path.display()))?;

    let mut count = 0;
    for workout in new_workouts {
        if existing_ids.contains(&workout.id) {
            continue;
        }
        let json = serde_json::to_string(workout).context("failed to serialize workout")?;
        writeln!(file, "{}", json).context("failed to write workout")?;
        count += 1;
    }

    Ok(count)
}

/// Check if stroke data exists for a given workout ID.
pub fn has_stroke_data(workout_id: u64) -> Result<bool> {
    Ok(strokes_path(workout_id)?.exists())
}

/// Write stroke data for a workout.
pub fn write_stroke_data(workout_id: u64, strokes: &[StrokeData]) -> Result<()> {
    let path = strokes_path(workout_id)?;
    let mut file = OpenOptions::new()
        .create(true)
        .write(true)
        .truncate(true)
        .open(&path)
        .with_context(|| format!("failed to open {}", path.display()))?;

    for stroke in strokes {
        let json = serde_json::to_string(stroke).context("failed to serialize stroke")?;
        writeln!(file, "{}", json).context("failed to write stroke")?;
    }

    Ok(())
}

/// Read stroke data for a workout.
pub fn read_stroke_data(workout_id: u64) -> Result<Vec<StrokeData>> {
    let path = strokes_path(workout_id)?;
    if !path.exists() {
        return Ok(Vec::new());
    }

    let file = fs::File::open(&path)
        .with_context(|| format!("failed to open {}", path.display()))?;
    let reader = BufReader::new(file);
    let mut strokes = Vec::new();

    for (i, line) in reader.lines().enumerate() {
        let line = line.with_context(|| format!("failed to read line {}", i + 1))?;
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        let stroke: StrokeData = serde_json::from_str(line)
            .with_context(|| format!("failed to parse stroke on line {}", i + 1))?;
        strokes.push(stroke);
    }

    Ok(strokes)
}

/// Get the total count of stored workouts (without parsing them all).
pub fn workout_count() -> Result<usize> {
    let path = workouts_path()?;
    if !path.exists() {
        return Ok(0);
    }
    let file = fs::File::open(&path)?;
    let reader = BufReader::new(file);
    Ok(reader.lines().filter(|l| l.as_ref().is_ok_and(|s| !s.trim().is_empty())).count())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::HeartRate;

    fn test_workout(id: u64) -> Workout {
        Workout {
            id,
            user_id: 1,
            date: "2026-03-02 17:41:00".to_string(),
            timezone: Some("America/Denver".to_string()),
            distance: 5500,
            machine_type: "rower".to_string(),
            time: 19122,
            time_formatted: "31:52.2".to_string(),
            workout_type: Some("JustRow".to_string()),
            weight_class: Some("H".to_string()),
            stroke_rate: Some(26),
            stroke_count: Some(832),
            calories_total: Some(280),
            drag_factor: Some(83),
            heart_rate: Some(HeartRate {
                average: Some(118),
                min: None,
                max: None,
            }),
            pace_500m: Some("2:53.8".to_string()),
            average_watts: Some(67),
            comments: None,
            updated_at: None,
        }
    }

    #[test]
    fn test_workout_serialization() {
        let w = test_workout(1);
        let json = serde_json::to_string(&w).unwrap();
        let parsed: Workout = serde_json::from_str(&json).unwrap();
        assert_eq!(parsed.id, 1);
        assert_eq!(parsed.distance, 5500);
    }
}
