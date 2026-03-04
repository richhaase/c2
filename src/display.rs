use crate::models::Workout;

/// Format meters with comma separators: 1000000 → "1,000,000"
pub fn format_meters(m: u64) -> String {
    let s = m.to_string();
    let mut result = String::new();
    for (i, c) in s.chars().rev().enumerate() {
        if i > 0 && i % 3 == 0 {
            result.push(',');
        }
        result.push(c);
    }
    result.chars().rev().collect()
}

/// Format a percentage with one decimal: 0.1234 → "12.3%"
pub fn format_percent(ratio: f64) -> String {
    format!("{:.1}%", ratio * 100.0)
}

/// Format a workout as a compact log line.
/// Example: "03/02  5500m  31:52.2  2:53.8/500m  26spm  118bpm  83df"
pub fn format_workout_line(w: &Workout, date_format: &str) -> String {
    let date_str = match w.parsed_date() {
        Some(dt) => dt.format(date_format).to_string(),
        None => w.date[..10].to_string(),
    };

    let distance = format!("{}m", format_meters(w.distance));
    let time = &w.time_formatted;
    let pace = w.pace_500m.as_deref().unwrap_or("-");
    let spm = w
        .stroke_rate
        .map(|s| format!("{}spm", s))
        .unwrap_or_else(|| "-".to_string());
    let hr = w
        .heart_rate
        .as_ref()
        .and_then(|h| h.average)
        .map(|h| format!("{}bpm", h))
        .unwrap_or_else(|| "-".to_string());
    let df = w
        .drag_factor
        .map(|d| format!("{}df", d))
        .unwrap_or_else(|| "-".to_string());

    format!(
        "{}  {}  {}  {}/500m  {}  {}  {}",
        date_str, distance, time, pace, spm, hr, df
    )
}

/// Format meters per week for display: 19231 → "19,231m/week"
pub fn format_meters_per_week(m: u64) -> String {
    format!("{}m/week", format_meters(m))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_format_meters() {
        assert_eq!(format_meters(0), "0");
        assert_eq!(format_meters(500), "500");
        assert_eq!(format_meters(1000), "1,000");
        assert_eq!(format_meters(1_000_000), "1,000,000");
    }

    #[test]
    fn test_format_percent() {
        assert_eq!(format_percent(0.0), "0.0%");
        assert_eq!(format_percent(0.5), "50.0%");
        assert_eq!(format_percent(1.0), "100.0%");
    }
}
