mod acl;
mod auth;
mod config;
mod server;
mod socks5;
mod token_store;
mod tokens;
mod tunnel;

use clap::Parser;
use config::Config;
use server::{router, AppState};
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use tracing::info;

#[tokio::main]
async fn main() {
    // rustls 0.23 requires an explicit crypto provider; install ring globally.
    rustls::crypto::ring::default_provider()
        .install_default()
        .expect("failed to install rustls crypto provider");

    tracing_subscriber::fmt::init();

    let config = Config::parse();
    let port = config.port;
    let socks5_addr = config.socks5_addr.clone();
    let controller_url = config.controller_url.clone();
    let tenant = config.tenant.clone();
    let connector_tunnel_addr = config.connector_tunnel_addr.clone();

    // Resolve the internal CA PEM bytes for connector TLS verification.
    let ca_pem: Arc<Vec<u8>> = Arc::new(load_ca_pem(&config));

    let state = AppState {
        config,
        pending: Arc::new(Mutex::new(HashMap::new())),
    };

    tokio::spawn(async move {
        let handler = move |req: socks5::ConnectRequest, mut stream: tokio::net::TcpStream| {
            let controller_url = controller_url.clone();
            let tenant = tenant.clone();
            let connector_tunnel_addr = connector_tunnel_addr.clone();
            let ca_pem = Arc::clone(&ca_pem);
            async move {
                // ── 1. Obtain a valid (possibly refreshed) device token ──────────
                let access_token =
                    match tokens::get_valid_token(&controller_url, &tenant).await {
                        Ok(t) => t,
                        Err(e) => {
                            tracing::warn!(
                                "[client acl precheck] {}:{} — {}",
                                req.destination,
                                req.port,
                                e
                            );
                            socks5::reply_error(&mut stream).await;
                            return;
                        }
                    };

                // ── 2. Client-side ACL pre-check (fast deny / split-tunnel) ──────
                let acl_resp = match acl::check_access(
                    &controller_url,
                    &access_token,
                    &req.destination,
                    req.port,
                )
                .await
                {
                    Ok(r) => r,
                    Err(e) => {
                        tracing::warn!(
                            "[client acl precheck] check-access failed for {}:{}: {}",
                            req.destination,
                            req.port,
                            e
                        );
                        socks5::reply_error(&mut stream).await;
                        return;
                    }
                };

                if !acl_resp.allowed {
                    info!(
                        "[client acl precheck] DENIED {}:{} reason={}",
                        req.destination, req.port, acl_resp.reason
                    );
                    socks5::reply_error(&mut stream).await;
                    return;
                }

                info!(
                    "[client acl precheck] GRANTED {}:{} resource_id={}",
                    req.destination, req.port, acl_resp.resource_id
                );

                // ── 3. Open TLS tunnel to connector ──────────────────────────────
                if connector_tunnel_addr.is_empty() {
                    tracing::warn!(
                        "CONNECTOR_TUNNEL_ADDR not set — cannot tunnel {}:{}",
                        req.destination,
                        req.port
                    );
                    socks5::reply_error(&mut stream).await;
                    return;
                }

                if ca_pem.is_empty() {
                    tracing::warn!(
                        "INTERNAL_CA_CERT / CA_CERT_PATH not set — cannot verify connector TLS"
                    );
                    socks5::reply_error(&mut stream).await;
                    return;
                }

                let mut tunnel_stream = match tunnel::open(
                    &connector_tunnel_addr,
                    &ca_pem,
                    &access_token,
                    &req.destination,
                    req.port,
                )
                .await
                {
                    Ok(s) => s,
                    Err(e) => {
                        tracing::warn!(
                            "tunnel open failed for {}:{}: {}",
                            req.destination,
                            req.port,
                            e
                        );
                        socks5::reply_error(&mut stream).await;
                        return;
                    }
                };

                // ── 4. Tell the SOCKS5 client the connection is established ──────
                if let Err(e) = socks5::reply_success(&mut stream).await {
                    tracing::warn!("reply_success failed: {}", e);
                    return;
                }

                // ── 5. Splice bytes: SOCKS5 client ↔ connector tunnel ────────────
                // The connector will do its own final ACL enforcement before
                // forwarding to the resource.
                match tokio::io::copy_bidirectional(&mut stream, &mut tunnel_stream).await {
                    Ok((sent, recv)) => info!(
                        "tunnel closed {}:{} sent={} recv={}",
                        req.destination, req.port, sent, recv
                    ),
                    Err(e) => tracing::warn!(
                        "tunnel I/O error {}:{}: {}",
                        req.destination, req.port, e
                    ),
                }
            }
        };

        if let Err(e) = socks5::listen(&socks5_addr, handler).await {
            tracing::warn!("SOCKS5 listener stopped: {}", e);
        }
    });

    let app = router(state);
    let addr = format!("127.0.0.1:{}", port);
    info!("ztna-client listening on http://{}", addr);

    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .expect("failed to bind");
    axum::serve(listener, app).await.expect("server failed");
}

/// Resolve the internal CA PEM bytes from config.
/// `internal_ca_cert` (env INTERNAL_CA_CERT) takes precedence over `ca_cert_path`.
fn load_ca_pem(config: &Config) -> Vec<u8> {
    if !config.internal_ca_cert.is_empty() {
        return config.internal_ca_cert.as_bytes().to_vec();
    }
    if !config.ca_cert_path.is_empty() {
        match std::fs::read(&config.ca_cert_path) {
            Ok(b) => return b,
            Err(e) => tracing::warn!("failed to read CA cert from {}: {}", config.ca_cert_path, e),
        }
    }
    Vec::new()
}
