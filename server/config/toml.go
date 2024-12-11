// Copyright 2024 Oraichain Labs
package config

// DefaultConfigTemplate defines the configuration template for the indexer RPC configuration
const DefaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml
###############################################################################
###                      Indexer Service Configuration                      ###
###############################################################################

[indexer-service]

# Enable defines if the RPC server should be enabled.
enable = {{ .IService.Enable }}

# EnableUnsafeCORS defines if CORS should be enabled (unsafe - use it at your own risk).
enabled-unsafe-cors = {{ .IService.EnableUnsafeCORS }}

# Address defines the RPC server address to bind to.
address = "{{ .IService.Address }}"

# HTTPTimeout is the read/write timeout of RPC server.
http-timeout = "{{ .IService.HTTPTimeout }}"

# HTTPIdleTimeout is the idle timeout of the RPC server.
http-idle-timeout = "{{ .IService.HTTPIdleTimeout }}"

# MaxOpenConnections sets the maximum number of simultaneous connections
# for the server listener.
max-open-connections = {{ .IService.MaxOpenConnections }}

# MetricsAddress defines the RPC Metrics server address to bind to. Pass --metrics in CLI to enable
# Prometheus metrics path: /debug/metrics/prometheus
metrics-address = "{{ .IService.MetricsAddress }}"
`
