use clap::Parser;

#[derive(Parser, Debug, Clone)]
#[command(name = "ztna-client", about = "ZTNA native client")]
pub struct Config {
    /// Controller URL (e.g. http://localhost:8081)
    #[arg(long, env = "CONTROLLER_URL", default_value = "http://localhost:8081")]
    pub controller_url: String,

    /// Local port to listen on
    #[arg(long, env = "ZTNA_CLIENT_PORT", default_value_t = 19515)]
    pub port: u16,

    /// Local SOCKS5 proxy address
    #[arg(long, env = "SOCKS5_ADDR", default_value = "127.0.0.1:1080")]
    pub socks5_addr: String,

    /// Active tenant slug (workspace) to use for access checks
    #[arg(long, env = "ZTNA_TENANT", default_value = "")]
    pub tenant: String,

    /// Connector device-tunnel address (host:port). Empty = no tunneling.
    #[arg(long, env = "CONNECTOR_TUNNEL_ADDR", default_value = "")]
    pub connector_tunnel_addr: String,

    /// Internal CA certificate PEM (inline). Used to verify the connector's TLS cert.
    /// Set either this or --ca-cert-path; this takes precedence.
    #[arg(long, env = "INTERNAL_CA_CERT", default_value = "")]
    pub internal_ca_cert: String,

    /// Path to the internal CA certificate PEM file.
    #[arg(long, env = "CA_CERT_PATH", default_value = "")]
    pub ca_cert_path: String,
}
