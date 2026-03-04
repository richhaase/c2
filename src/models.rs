use chrono::NaiveDateTime;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserProfile {
    pub id: u64,
    pub username: String,
    #[serde(default)]
    pub first_name: Option<String>,
    #[serde(default)]
    pub last_name: Option<String>,
    #[serde(default)]
    pub email: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HeartRate {
    #[serde(default)]
    pub average: Option<u16>,
    #[serde(default)]
    pub min: Option<u16>,
    #[serde(default)]
    pub max: Option<u16>,
}

/// A single workout result from the Concept2 Logbook API.
/// The `time` field is in tenths of a second (e.g., 19122 = 31:52.2).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Workout {
    pub id: u64,
    pub user_id: u64,
    pub date: String,
    #[serde(default)]
    pub timezone: Option<String>,
    pub distance: u64,
    #[serde(rename = "type")]
    pub machine_type: String,
    /// Duration in tenths of a second
    pub time: u64,
    pub time_formatted: String,
    #[serde(default)]
    pub workout_type: Option<String>,
    #[serde(default)]
    pub weight_class: Option<String>,
    #[serde(default)]
    pub stroke_rate: Option<u16>,
    #[serde(default)]
    pub stroke_count: Option<u32>,
    #[serde(default)]
    pub calories_total: Option<u32>,
    #[serde(default)]
    pub drag_factor: Option<u16>,
    #[serde(default)]
    pub heart_rate: Option<HeartRate>,
    #[serde(default)]
    pub pace_500m: Option<String>,
    #[serde(default)]
    pub average_watts: Option<u32>,
    #[serde(default)]
    pub comments: Option<String>,
    #[serde(default)]
    pub updated_at: Option<String>,
}

impl Workout {
    /// Parse the date string ("YYYY-MM-DD HH:MM:SS") into a NaiveDateTime.
    pub fn parsed_date(&self) -> Option<NaiveDateTime> {
        NaiveDateTime::parse_from_str(&self.date, "%Y-%m-%d %H:%M:%S").ok()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StrokeData {
    #[serde(default)]
    pub time: Option<f64>,
    #[serde(default)]
    pub distance: Option<f64>,
    #[serde(default)]
    pub pace: Option<f64>,
    #[serde(rename = "strokesPerMinute", alias = "spm", default)]
    pub spm: Option<u16>,
    #[serde(rename = "heartRate", alias = "hr", default)]
    pub heart_rate: Option<u16>,
}

/// Paginated response wrapper from the API.
#[derive(Debug, Clone, Deserialize)]
pub struct PaginatedResponse<T> {
    pub data: Vec<T>,
    #[serde(default)]
    pub meta: Option<PaginationMeta>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct PaginationMeta {
    #[serde(default)]
    pub current_page: Option<u32>,
    #[serde(default)]
    pub last_page: Option<u32>,
    #[serde(default)]
    pub total: Option<u32>,
}
