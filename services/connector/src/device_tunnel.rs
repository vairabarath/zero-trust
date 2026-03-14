//! Device-tunnel endpoint for ztna-client connections.
//!
//! Wire protocol (TLS-wrapped TCP):
//!   1. Client completes TLS handshake (connector's SPIFFE cert, no client cert required).
//!   2. Client sends a newline-terminated JSON line:
//!        {"token":"<device_jwt>","destination":"host","port":443}
//!   3. Server replies with a newline-terminated JSON line:
//!        {"ok":true}   — then raw bytes flow in both directions
//!        {"ok":false,"error":"reason"}  — then server closes
use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};
use tokio::net::{TcpListener, TcpStream};
use tokio_rustls::TlsAcceptor;
use tracing::{info, warn};

use crate::tls::cert_store::CertStore;
use crate::tls::server_cfg::build_device_tunnel_tls;

#[derive(Deserialize)]
struct TunnelRequest {
    token: String,
    destination: String,
    port: u16,
}

#[derive(Serialize)]
struct TunnelResponse {
    ok: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

#[derive(Deserialize)]
struct CheckAccessResponse {
    allowed: bool,
}

pub async fn listen(addr: &str, controller_http_url: String, store: CertStore) -> Result<()> {
    let tls_config = build_device_tunnel_tls(&store)?;
    let acceptor = TlsAcceptor::from(Arc::new(tls_config));

    let listener = TcpListener::bind(addr).await?;
    info!("[connector acl enforcement] device tunnel (TLS) listening on {}", addr);

    loop {
        match listener.accept().await {
            Ok((stream, peer)) => {
                let ctrl = controller_http_url.clone();
                let acc = acceptor.clone();
                tokio::spawn(async move {
                    match acc.accept(stream).await {
                        Ok(tls) => {
                            if let Err(e) = handle(tls, &ctrl).await {
                                warn!(
                                    "[connector acl enforcement] client error from {}: {}",
                                    peer, e
                                );
                            }
                        }
                        Err(e) => warn!(
                            "[connector acl enforcement] TLS accept from {}: {}",
                            peer, e
                        ),
                    }
                });
            }
            Err(e) => warn!("[connector acl enforcement] accept error: {}", e),
        }
    }
}

/// Read bytes one-at-a-time until `\n`, up to 4 KiB.
async fn read_line<S: AsyncRead + Unpin>(stream: &mut S) -> Result<String> {
    let mut buf = Vec::with_capacity(256);
    let mut byte = [0u8; 1];
    loop {
        let n = stream.read(&mut byte).await?;
        if n == 0 {
            anyhow::bail!("EOF before handshake newline");
        }
        if byte[0] == b'\n' {
            break;
        }
        buf.push(byte[0]);
        if buf.len() > 4096 {
            anyhow::bail!("handshake line too long");
        }
    }
    Ok(String::from_utf8(buf)?)
}

async fn send_response<S: AsyncWrite + Unpin>(
    stream: &mut S,
    ok: bool,
    error: Option<&str>,
) -> Result<()> {
    let resp = TunnelResponse { ok, error: error.map(|s| s.to_string()) };
    let mut line = serde_json::to_string(&resp)?;
    line.push('\n');
    stream.write_all(line.as_bytes()).await?;
    Ok(())
}

async fn handle(
    mut stream: tokio_rustls::server::TlsStream<TcpStream>,
    controller_http_url: &str,
) -> Result<()> {
    let line = read_line(&mut stream).await?;
    let req: TunnelRequest = serde_json::from_str(line.trim())
        .map_err(|e| anyhow::anyhow!("bad handshake: {}", e))?;

    let ok = check_access(controller_http_url, &req.token, &req.destination, req.port).await;

    match ok {
        Err(e) => {
            let _ = send_response(&mut stream, false, Some("check-access error")).await;
            return Err(e);
        }
        Ok(false) => {
            send_response(&mut stream, false, Some("access denied")).await?;
            info!(
                "[connector acl enforcement] DENIED: {}:{}",
                req.destination, req.port
            );
            return Ok(());
        }
        Ok(true) => {}
    }

    // Open TCP to the resource
    let dest = format!("{}:{}", req.destination, req.port);
    let mut resource = match TcpStream::connect(&dest).await {
        Ok(s) => s,
        Err(e) => {
            let msg = format!("connect to {} failed: {}", dest, e);
            let _ = send_response(&mut stream, false, Some(&msg)).await;
            return Err(anyhow::anyhow!("{}", msg));
        }
    };

    send_response(&mut stream, true, None).await?;
    info!("[connector acl enforcement] OPEN: {}", dest);

    match tokio::io::copy_bidirectional(&mut stream, &mut resource).await {
        Ok((sent, recv)) => info!(
            "[connector acl enforcement] CLOSED: {} sent={} recv={}",
            dest, sent, recv
        ),
        Err(e) => warn!("[connector acl enforcement] I/O error {}: {}", dest, e),
    }

    Ok(())
}

async fn check_access(
    controller_http_url: &str,
    token: &str,
    destination: &str,
    port: u16,
) -> Result<bool> {
    #[derive(Serialize)]
    struct Req<'a> {
        destination: &'a str,
        protocol: &'a str,
        port: u16,
    }

    let resp = reqwest::Client::new()
        .post(format!("{}/api/device/check-access", controller_http_url))
        .bearer_auth(token)
        .json(&Req { destination, protocol: "tcp", port })
        .send()
        .await?;

    if !resp.status().is_success() {
        let text = resp.text().await.unwrap_or_default();
        anyhow::bail!("check-access: {}", text);
    }

    let body: CheckAccessResponse = resp.json().await?;
    Ok(body.allowed)
}
