package config

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "github.com/BurntSushi/toml"
    "gopkg.in/yaml.v3"
)

type TLSConfig struct {
    Enabled bool   `json:"enabled" yaml:"enabled" toml:"enabled"`
    CertFile string `json:"cert_file" yaml:"cert_file" toml:"cert_file"`
    KeyFile string `json:"key_file" yaml:"key_file" toml:"key_file"`
}

type ClientTLSConfig struct {
    Enabled bool `json:"enabled" yaml:"enabled" toml:"enabled"`
    InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify" toml:"insecure_skip_verify"`
}

type PortRange struct {
    Min int `json:"min" yaml:"min" toml:"min"`
    Max int `json:"max" yaml:"max" toml:"max"`
}

type ServerConfig struct {
    Name string `json:"name" yaml:"name" toml:"name"`
    ListenIP string `json:"listen_ip" yaml:"listen_ip" toml:"listen_ip"`
    PortRange PortRange `json:"port_range" yaml:"port_range" toml:"port_range"`
    Protocol string `json:"protocol" yaml:"protocol" toml:"protocol"`
    TOTPSecret string `json:"totp_secret" yaml:"totp_secret" toml:"totp_secret"`
    StepSeconds int `json:"step_seconds" yaml:"step_seconds" toml:"step_seconds"`
    SkewSteps int `json:"skew_steps" yaml:"skew_steps" toml:"skew_steps"`
    TargetAddr string `json:"target_addr" yaml:"target_addr" toml:"target_addr"`
    TargetPort int `json:"target_port" yaml:"target_port" toml:"target_port"`
    AllowedCIDRs []string `json:"allowed_client_ips" yaml:"allowed_client_ips" toml:"allowed_client_ips"`
    TLS TLSConfig `json:"tls" yaml:"tls" toml:"tls"`
}

type ClientConfig struct {
    Name string `json:"name" yaml:"name" toml:"name"`
    ServerHost string `json:"server_host" yaml:"server_host" toml:"server_host"`
    PortRange PortRange `json:"port_range" yaml:"port_range" toml:"port_range"`
    Protocol string `json:"protocol" yaml:"protocol" toml:"protocol"`
    TOTPSecret string `json:"totp_secret" yaml:"totp_secret" toml:"totp_secret"`
    StepSeconds int `json:"step_seconds" yaml:"step_seconds" toml:"step_seconds"`
    SkewSteps int `json:"skew_steps" yaml:"skew_steps" toml:"skew_steps"`
    BindIP string `json:"bind_ip" yaml:"bind_ip" toml:"bind_ip"`
    BindPort int `json:"bind_port" yaml:"bind_port" toml:"bind_port"`
    ClientID string `json:"client_id" yaml:"client_id" toml:"client_id"`
    TLS ClientTLSConfig `json:"tls" yaml:"tls" toml:"tls"`
}

type MultiServerConfig struct {
    Routes []ServerConfig `json:"routes" yaml:"routes" toml:"routes"`
}

type MultiClientConfig struct {
    Endpoints []ClientConfig `json:"endpoints" yaml:"endpoints" toml:"endpoints"`
}

func LoadServerConfig(path string) (ServerConfig, error) {
    var c ServerConfig
    b, err := os.ReadFile(path)
    if err != nil {
        return c, err
    }
    if err := unmarshalByExt(b, path, &c); err != nil {
        return c, err
    }
    if c.PortRange.Min <= 0 || c.PortRange.Max <= 0 || c.PortRange.Min > c.PortRange.Max {
        return c, errors.New("invalid port_range")
    }
    if c.Protocol != "tcp" && c.Protocol != "udp" {
        return c, errors.New("invalid protocol")
    }
    if c.StepSeconds <= 0 {
        return c, errors.New("invalid step_seconds")
    }
    if c.TargetAddr == "" || c.TargetPort <= 0 {
        return c, errors.New("invalid target")
    }
    return c, nil
}

func LoadServerConfigs(path string) ([]ServerConfig, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var multi MultiServerConfig
    if err := unmarshalByExt(b, path, &multi); err == nil && len(multi.Routes) > 0 {
        for i := range multi.Routes {
            if _, err := json.Marshal(multi.Routes[i]); err != nil {
                return nil, err
            }
            // validate each route
            if _, err := validateServerConfig(&multi.Routes[i]); err != nil {
                return nil, err
            }
        }
        // check port range overlap among routes
        for i := 0; i < len(multi.Routes); i++ {
            for j := i + 1; j < len(multi.Routes); j++ {
                if overlap(multi.Routes[i].PortRange, multi.Routes[j].PortRange) {
                    return nil, errors.New("server routes port_range overlap detected")
                }
            }
        }
        return multi.Routes, nil
    }
    // fallback single
    c, err := LoadServerConfig(path)
    if err != nil {
        return nil, err
    }
    return []ServerConfig{c}, nil
}

func validateServerConfig(c *ServerConfig) (ServerConfig, error) {
    if c.PortRange.Min <= 0 || c.PortRange.Max <= 0 || c.PortRange.Min > c.PortRange.Max {
        return *c, errors.New("invalid port_range")
    }
    if c.Protocol != "tcp" && c.Protocol != "udp" {
        return *c, errors.New("invalid protocol")
    }
    if c.StepSeconds <= 0 {
        return *c, errors.New("invalid step_seconds")
    }
    if c.TargetAddr == "" || c.TargetPort <= 0 {
        return *c, errors.New("invalid target")
    }
    return *c, nil
}

func LoadClientConfig(path string) (ClientConfig, error) {
    var c ClientConfig
    b, err := os.ReadFile(path)
    if err != nil {
        return c, err
    }
    if err := unmarshalByExt(b, path, &c); err != nil {
        return c, err
    }
    if c.PortRange.Min <= 0 || c.PortRange.Max <= 0 || c.PortRange.Min > c.PortRange.Max {
        return c, errors.New("invalid port_range")
    }
    if c.Protocol != "tcp" && c.Protocol != "udp" {
        return c, errors.New("invalid protocol")
    }
    if c.StepSeconds <= 0 {
        return c, errors.New("invalid step_seconds")
    }
    if c.BindPort <= 0 {
        return c, errors.New("invalid bind_port")
    }
    if c.ServerHost == "" {
        return c, errors.New("invalid server_host")
    }
    if c.ClientID == "" {
        c.ClientID = "client"
    }
    return c, nil
}

func LoadClientConfigs(path string) ([]ClientConfig, error) {
    b, err := os.ReadFile(path)
    if err != nil { return nil, err }
    var multi MultiClientConfig
    if err := unmarshalByExt(b, path, &multi); err == nil && len(multi.Endpoints) > 0 {
        for i := range multi.Endpoints {
            if _, err := validateClientConfig(&multi.Endpoints[i]); err != nil { return nil, err }
        }
        // check local bind duplicates
        seen := map[string]struct{}{}
        for _, ep := range multi.Endpoints {
            key := ep.BindIP + ":" + strconv.Itoa(ep.BindPort)
            if _, ok := seen[key]; ok {
                return nil, errors.New("client endpoints bind_ip:bind_port duplicated")
            }
            seen[key] = struct{}{}
        }
        return multi.Endpoints, nil
    }
    c, err := LoadClientConfig(path)
    if err != nil { return nil, err }
    return []ClientConfig{c}, nil
}

func validateClientConfig(c *ClientConfig) (ClientConfig, error) {
    if c.PortRange.Min <= 0 || c.PortRange.Max <= 0 || c.PortRange.Min > c.PortRange.Max {
        return *c, errors.New("invalid port_range")
    }
    if c.Protocol != "tcp" && c.Protocol != "udp" {
        return *c, errors.New("invalid protocol")
    }
    if c.StepSeconds <= 0 {
        return *c, errors.New("invalid step_seconds")
    }
    if c.BindPort <= 0 {
        return *c, errors.New("invalid bind_port")
    }
    if c.ServerHost == "" {
        return *c, errors.New("invalid server_host")
    }
    if c.ClientID == "" { c.ClientID = "client" }
    return *c, nil
}

func overlap(a, b PortRange) bool {
    if a.Max < a.Min || b.Max < b.Min { return false }
    return !(a.Max < b.Min || b.Max < a.Min)
}

func unmarshalByExt(b []byte, path string, v interface{}) error {
    ext := strings.ToLower(filepath.Ext(path))
    switch ext {
    case ".yaml", ".yml":
        return yaml.Unmarshal(b, v)
    case ".toml":
        return toml.Unmarshal(b, v)
    default:
        return json.Unmarshal(b, v)
    }
}