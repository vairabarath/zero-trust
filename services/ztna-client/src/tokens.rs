//! Token lifecycle: load stored device tokens and refresh when near expiry.
use anyhow::Result;
use std::time::{SystemTime, UNIX_EPOCH};
use tracing::{info, warn};

use crate::auth::refresh_device_token;
use crate::token_store::{load_tokens, save_tokens, StoredTokens};

/// Refresh if the token expires within this many seconds.
const REFRESH_BUFFER_SECS: i64 = 60;

fn now_unix() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs() as i64
}

/// Return a valid access token for `tenant`.
///
/// - If no tokens are stored, returns `Err` with a "no active session" message.
/// - If the token is expired or within `REFRESH_BUFFER_SECS` of expiry, refreshes
///   it via the controller and persists the new tokens.
/// - If refresh fails, returns `Err`.
pub async fn get_valid_token(controller_url: &str, tenant: &str) -> Result<String> {
    let mut stored = load_tokens(tenant)
        .ok_or_else(|| anyhow::anyhow!("no active session for tenant '{}'", tenant))?;

    let now = now_unix();
    if now >= stored.expires_at - REFRESH_BUFFER_SECS {
        info!(
            "device token near expiry (expires_at={}, now={}), refreshing (tenant={})",
            stored.expires_at, now, tenant
        );

        let refreshed = refresh_device_token(controller_url, &stored.refresh_token)
            .await
            .map_err(|e| anyhow::anyhow!("token refresh failed for '{}': {}", tenant, e))?;

        stored = StoredTokens {
            access_token: refreshed.access_token,
            refresh_token: refreshed.refresh_token,
            workspace_name: stored.workspace_name,
            workspace_slug: stored.workspace_slug,
            expires_at: now + refreshed.expires_in,
        };

        if let Err(e) = save_tokens(tenant, &stored) {
            warn!("failed to persist refreshed token for '{}': {}", tenant, e);
        }
    }

    Ok(stored.access_token)
}
