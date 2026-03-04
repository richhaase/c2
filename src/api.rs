use anyhow::{bail, Context, Result};
use reqwest::Client;

use crate::config::{self, Tokens};
use crate::models::{PaginatedResponse, StrokeData, UserProfile, Workout};

pub struct ApiClient {
    client: Client,
    base_url: String,
    tokens: Tokens,
}

impl ApiClient {
    pub fn new(base_url: &str, tokens: Tokens) -> Result<Self> {
        let client = Client::builder()
            .user_agent("c2cli/0.1.0")
            .build()
            .context("failed to build HTTP client")?;

        Ok(Self {
            client,
            base_url: base_url.trim_end_matches('/').to_string(),
            tokens,
        })
    }

    /// Load an ApiClient from stored config and tokens.
    pub fn from_config() -> Result<Self> {
        let cfg = config::load_config()?;
        let tokens = config::load_tokens()?;
        Self::new(&cfg.api.base_url, tokens)
    }

    /// Make an authenticated GET request. Retries once on 401 after refreshing tokens.
    async fn get(&mut self, path: &str) -> Result<reqwest::Response> {
        let url = format!("{}{}", self.base_url, path);
        let resp = self
            .client
            .get(&url)
            .bearer_auth(&self.tokens.access_token)
            .send()
            .await
            .with_context(|| format!("request to {} failed", url))?;

        if resp.status() == reqwest::StatusCode::UNAUTHORIZED {
            self.refresh_token().await?;
            let resp = self
                .client
                .get(&url)
                .bearer_auth(&self.tokens.access_token)
                .send()
                .await
                .with_context(|| format!("retry request to {} failed", url))?;
            Ok(resp)
        } else {
            Ok(resp)
        }
    }

    /// Refresh the access token using the refresh token.
    async fn refresh_token(&mut self) -> Result<()> {
        let cfg = config::load_config()?;
        let url = format!("{}/oauth/access_token", self.base_url);

        let resp = self
            .client
            .post(&url)
            .form(&[
                ("grant_type", "refresh_token"),
                ("client_id", &cfg.api.client_id),
                ("client_secret", &cfg.api.client_secret),
                ("refresh_token", &self.tokens.refresh_token),
            ])
            .send()
            .await
            .context("token refresh request failed")?;

        if !resp.status().is_success() {
            let status = resp.status();
            let body = resp.text().await.unwrap_or_default();
            bail!("token refresh failed ({}): {}", status, body);
        }

        let new_tokens: Tokens = resp.json().await.context("failed to parse token response")?;
        config::save_tokens(&new_tokens)?;
        self.tokens = new_tokens;
        Ok(())
    }

    /// Get the authenticated user's profile.
    pub async fn get_user(&mut self) -> Result<UserProfile> {
        let resp = self.get("/api/users/me").await?;
        let status = resp.status();
        if !status.is_success() {
            let body = resp.text().await.unwrap_or_default();
            bail!("get user failed ({}): {}", status, body);
        }
        let profile: UserProfile = resp.json().await.context("failed to parse user profile")?;
        Ok(profile)
    }

    /// Fetch workout results with optional date filters and pagination.
    pub async fn get_results(
        &mut self,
        from: Option<&str>,
        to: Option<&str>,
        page: u32,
    ) -> Result<PaginatedResponse<Workout>> {
        let mut path = format!("/api/users/me/results?type=rower&page={}", page);
        if let Some(from) = from {
            path.push_str(&format!("&from={}", from));
        }
        if let Some(to) = to {
            path.push_str(&format!("&to={}", to));
        }

        let resp = self.get(&path).await?;
        let status = resp.status();
        if !status.is_success() {
            let body = resp.text().await.unwrap_or_default();
            bail!("get results failed ({}): {}", status, body);
        }
        let results: PaginatedResponse<Workout> =
            resp.json().await.context("failed to parse results")?;
        Ok(results)
    }

    /// Fetch all workout results with automatic pagination.
    pub async fn get_all_results(
        &mut self,
        from: Option<&str>,
        to: Option<&str>,
    ) -> Result<Vec<Workout>> {
        let mut all_workouts = Vec::new();
        let mut page = 1u32;

        loop {
            let response = self.get_results(from, to, page).await?;
            let count = response.data.len();
            all_workouts.extend(response.data);

            let has_more = response
                .meta
                .as_ref()
                .and_then(|m| m.last_page.zip(m.current_page))
                .is_some_and(|(last, current)| current < last);

            if !has_more || count == 0 {
                break;
            }
            page += 1;
        }

        Ok(all_workouts)
    }

    /// Fetch stroke data for a specific workout.
    pub async fn get_strokes(&mut self, workout_id: u64) -> Result<Vec<StrokeData>> {
        let path = format!("/api/users/me/results/{}/strokes", workout_id);
        let resp = self.get(&path).await?;
        let status = resp.status();
        if !status.is_success() {
            // Not all workouts have stroke data — 404 is fine
            if status == reqwest::StatusCode::NOT_FOUND {
                return Ok(Vec::new());
            }
            let body = resp.text().await.unwrap_or_default();
            bail!("get strokes failed ({}): {}", status, body);
        }
        let strokes: Vec<StrokeData> = resp.json().await.context("failed to parse strokes")?;
        Ok(strokes)
    }
}
