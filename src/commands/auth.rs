use anyhow::{bail, Context, Result};
use tokio::io::AsyncBufReadExt;
use tokio::net::TcpListener;

use crate::api::ApiClient;
use crate::config::{self, Tokens};

pub async fn run() -> Result<()> {
    let cfg = config::load_config()?;
    config::ensure_dirs()?;

    let auth_url = format!(
        "{}/oauth/authorize?client_id={}&redirect_uri={}&response_type=code&scope=user:read,results:read",
        cfg.api.base_url, cfg.api.client_id, cfg.api.redirect_uri
    );

    println!("Opening browser for Concept2 authorization...");
    println!();
    println!("If the browser doesn't open, visit this URL:");
    println!("{}", auth_url);
    println!();

    // Try to open browser
    let _ = open::that(&auth_url);

    // Start local server to catch the redirect
    let listener = TcpListener::bind("127.0.0.1:9876")
        .await
        .context("failed to bind to port 9876 — is another instance running?")?;

    println!("Waiting for authorization callback on http://localhost:9876/callback ...");

    let (stream, _) = listener.accept().await.context("failed to accept connection")?;
    let mut reader = tokio::io::BufReader::new(stream);
    let mut request_line = String::new();
    reader
        .read_line(&mut request_line)
        .await
        .context("failed to read request")?;

    // Extract the authorization code from the callback URL
    let code = extract_code(&request_line)?;

    // Send a response to the browser
    let response_body = "Authorization successful! You can close this tab.";
    let response = format!(
        "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: {}\r\n\r\n{}",
        response_body.len(),
        response_body
    );
    let inner = reader.into_inner();
    let mut write_stream = inner;
    tokio::io::AsyncWriteExt::write_all(&mut write_stream, response.as_bytes())
        .await
        .context("failed to send response")?;

    println!("Authorization code received. Exchanging for tokens...");

    // Exchange code for tokens
    let client = reqwest::Client::new();
    let token_url = format!("{}/oauth/access_token", cfg.api.base_url);
    let resp = client
        .post(&token_url)
        .form(&[
            ("grant_type", "authorization_code"),
            ("client_id", cfg.api.client_id.as_str()),
            ("client_secret", cfg.api.client_secret.as_str()),
            ("redirect_uri", cfg.api.redirect_uri.as_str()),
            ("code", &code),
        ])
        .send()
        .await
        .context("token exchange request failed")?;

    if !resp.status().is_success() {
        let status = resp.status();
        let body = resp.text().await.unwrap_or_default();
        bail!("token exchange failed ({}): {}", status, body);
    }

    let tokens: Tokens = resp.json().await.context("failed to parse token response")?;
    config::save_tokens(&tokens)?;
    println!("Tokens saved.");

    // Verify by fetching user profile
    let mut api = ApiClient::new(&cfg.api.base_url, tokens)?;
    let user = api.get_user().await?;
    println!();
    println!(
        "Authenticated as: {} (ID: {})",
        user.username, user.id
    );

    Ok(())
}

fn extract_code(request_line: &str) -> Result<String> {
    // Request line looks like: GET /callback?code=XXXXX HTTP/1.1
    let path = request_line
        .split_whitespace()
        .nth(1)
        .context("malformed HTTP request")?;

    let url = url::Url::parse(&format!("http://localhost{}", path))
        .context("failed to parse callback URL")?;

    for (key, value) in url.query_pairs() {
        if key == "code" {
            return Ok(value.to_string());
        }
    }

    bail!("no authorization code found in callback URL")
}
