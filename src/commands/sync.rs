use anyhow::Result;

use crate::api::ApiClient;
use crate::config;
use crate::storage;

pub async fn run() -> Result<()> {
    config::ensure_dirs()?;
    let cfg = config::load_config()?;
    let tokens = config::load_tokens()?;
    let mut api = ApiClient::new(&cfg.api.base_url, tokens)?;

    let from = cfg.sync.last_sync.as_deref();
    if let Some(since) = from {
        println!("Syncing workouts since {}...", since);
    } else {
        println!("First sync — pulling all workouts...");
    }

    let workouts = api.get_all_results(from, None).await?;
    let fetched = workouts.len();
    let written = storage::append_workouts(&workouts)?;

    println!("Fetched {} workouts, {} new.", fetched, written);

    // Fetch stroke data for new workouts
    let mut stroke_count = 0;
    for w in &workouts {
        if !storage::has_stroke_data(w.id)? {
            let strokes = api.get_strokes(w.id).await?;
            if !strokes.is_empty() {
                storage::write_stroke_data(w.id, &strokes)?;
                stroke_count += 1;
            }
        }
    }
    if stroke_count > 0 {
        println!("Fetched stroke data for {} workouts.", stroke_count);
    }

    // Update last_sync timestamp
    let mut cfg = cfg;
    cfg.sync.last_sync = Some(chrono::Utc::now().format("%Y-%m-%dT%H:%M:%SZ").to_string());
    config::save_config(&cfg)?;

    let total = storage::workout_count()?;
    println!("Total workouts: {}", total);

    Ok(())
}
