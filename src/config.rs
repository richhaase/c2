use anyhow::{Context, Result};
use chrono::NaiveDate;
use serde::{Deserialize, Serialize};
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub api: ApiConfig,
    #[serde(default)]
    pub sync: SyncConfig,
    #[serde(default)]
    pub goal: GoalConfig,
    #[serde(default)]
    pub display: DisplayConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiConfig {
    pub client_id: String,
    pub client_secret: String,
    #[serde(default = "default_redirect_uri")]
    pub redirect_uri: String,
    #[serde(default = "default_base_url")]
    pub base_url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct SyncConfig {
    #[serde(default)]
    pub last_sync: Option<String>,
    #[serde(default = "default_machine_type")]
    pub machine_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GoalConfig {
    #[serde(default = "default_target_meters")]
    pub target_meters: u64,
    #[serde(default)]
    pub start_date: Option<NaiveDate>,
    #[serde(default)]
    pub end_date: Option<NaiveDate>,
}

impl Default for GoalConfig {
    fn default() -> Self {
        Self {
            target_meters: default_target_meters(),
            start_date: None,
            end_date: None,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DisplayConfig {
    #[serde(default = "default_date_format")]
    pub date_format: String,
}

impl Default for DisplayConfig {
    fn default() -> Self {
        Self {
            date_format: default_date_format(),
        }
    }
}

fn default_redirect_uri() -> String {
    "http://localhost:9876/callback".to_string()
}

fn default_base_url() -> String {
    "https://log.concept2.com".to_string()
}

fn default_machine_type() -> String {
    "rower".to_string()
}

fn default_target_meters() -> u64 {
    1_000_000
}

fn default_date_format() -> String {
    "%m/%d".to_string()
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Tokens {
    pub access_token: String,
    pub refresh_token: String,
    #[serde(default)]
    pub expires_at: Option<i64>,
    #[serde(default)]
    pub token_type: Option<String>,
}

/// Returns the base config directory: ~/.config/c2cli/
pub fn config_dir() -> Result<PathBuf> {
    let home = dirs::home_dir().context("could not determine home directory")?;
    Ok(home.join(".config").join("c2cli"))
}

/// Returns the data directory: ~/.config/c2cli/data/
pub fn data_dir() -> Result<PathBuf> {
    Ok(config_dir()?.join("data"))
}

/// Ensure all required directories exist.
pub fn ensure_dirs() -> Result<()> {
    let config = config_dir()?;
    let data = data_dir()?;
    let strokes = data.join("strokes");
    fs::create_dir_all(&config).context("failed to create config directory")?;
    fs::create_dir_all(&data).context("failed to create data directory")?;
    fs::create_dir_all(&strokes).context("failed to create strokes directory")?;
    Ok(())
}

/// Load config from ~/.config/c2cli/config.toml
pub fn load_config() -> Result<Config> {
    let path = config_dir()?.join("config.toml");
    let contents = fs::read_to_string(&path)
        .with_context(|| format!("failed to read config at {}", path.display()))?;
    let config: Config = toml::from_str(&contents).context("failed to parse config.toml")?;
    Ok(config)
}

/// Save config to ~/.config/c2cli/config.toml
pub fn save_config(config: &Config) -> Result<()> {
    let path = config_dir()?.join("config.toml");
    let contents = toml::to_string_pretty(config).context("failed to serialize config")?;
    fs::write(&path, contents).context("failed to write config.toml")?;
    Ok(())
}

/// Load tokens from ~/.config/c2cli/tokens.json
pub fn load_tokens() -> Result<Tokens> {
    let path = config_dir()?.join("tokens.json");
    let contents = fs::read_to_string(&path)
        .with_context(|| format!("failed to read tokens at {}", path.display()))?;
    let tokens: Tokens = serde_json::from_str(&contents).context("failed to parse tokens.json")?;
    Ok(tokens)
}

/// Save tokens to ~/.config/c2cli/tokens.json with 0600 permissions.
pub fn save_tokens(tokens: &Tokens) -> Result<()> {
    let path = config_dir()?.join("tokens.json");
    let contents = serde_json::to_string_pretty(tokens).context("failed to serialize tokens")?;
    fs::write(&path, &contents).context("failed to write tokens.json")?;
    set_restrictive_permissions(&path)?;
    Ok(())
}

fn set_restrictive_permissions(path: &Path) -> Result<()> {
    let perms = fs::Permissions::from_mode(0o600);
    fs::set_permissions(path, perms).context("failed to set file permissions to 0600")?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config_roundtrip() {
        let config = Config {
            api: ApiConfig {
                client_id: "test_id".to_string(),
                client_secret: "test_secret".to_string(),
                redirect_uri: default_redirect_uri(),
                base_url: default_base_url(),
            },
            sync: SyncConfig::default(),
            goal: GoalConfig::default(),
            display: DisplayConfig::default(),
        };

        let serialized = toml::to_string_pretty(&config).unwrap();
        let deserialized: Config = toml::from_str(&serialized).unwrap();
        assert_eq!(deserialized.api.client_id, "test_id");
        assert_eq!(deserialized.goal.target_meters, 1_000_000);
    }

    #[test]
    fn test_tokens_roundtrip() {
        let tokens = Tokens {
            access_token: "abc".to_string(),
            refresh_token: "def".to_string(),
            expires_at: Some(1234567890),
            token_type: Some("Bearer".to_string()),
        };

        let json = serde_json::to_string(&tokens).unwrap();
        let parsed: Tokens = serde_json::from_str(&json).unwrap();
        assert_eq!(parsed.access_token, "abc");
        assert_eq!(parsed.refresh_token, "def");
    }
}
