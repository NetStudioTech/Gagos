package tools

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"time"
)

// CertInfo represents certificate information
type CertInfo struct {
	Subject            string    `json:"subject"`
	Issuer             string    `json:"issuer"`
	SerialNumber       string    `json:"serial_number"`
	NotBefore          time.Time `json:"not_before"`
	NotAfter           time.Time `json:"not_after"`
	DaysUntilExpiry    int       `json:"days_until_expiry"`
	IsExpired          bool      `json:"is_expired"`
	IsCA               bool      `json:"is_ca"`
	DNSNames           []string  `json:"dns_names,omitempty"`
	IPAddresses        []string  `json:"ip_addresses,omitempty"`
	SignatureAlgorithm string    `json:"signature_algorithm"`
	PublicKeyAlgorithm string    `json:"public_key_algorithm"`
	Version            int       `json:"version"`
	FingerprintSHA1    string    `json:"fingerprint_sha1"`
	FingerprintSHA256  string    `json:"fingerprint_sha256"`
}

// CertChainResult represents the certificate chain
type CertChainResult struct {
	Host         string     `json:"host"`
	Port         int        `json:"port"`
	Protocol     string     `json:"protocol,omitempty"`
	Certificates []CertInfo `json:"certificates"`
	ChainValid   bool       `json:"chain_valid"`
	Error        string     `json:"error,omitempty"`
}

// GetCertificateInfo fetches certificate info from a host
func GetCertificateInfo(host string, port int, timeout time.Duration) CertChainResult {
	if port == 0 {
		port = 443
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: timeout},
		"tcp",
		addr,
		&tls.Config{
			InsecureSkipVerify: true, // We want to see the cert even if invalid
		},
	)
	if err != nil {
		return CertChainResult{
			Host:  host,
			Port:  port,
			Error: "Failed to connect: " + err.Error(),
		}
	}
	defer conn.Close()

	state := conn.ConnectionState()
	certs := state.PeerCertificates

	result := CertChainResult{
		Host:         host,
		Port:         port,
		Protocol:     getTLSVersionName(state.Version),
		Certificates: make([]CertInfo, 0, len(certs)),
		ChainValid:   true,
	}

	for _, cert := range certs {
		info := parseCertificate(cert)
		result.Certificates = append(result.Certificates, info)

		if info.IsExpired {
			result.ChainValid = false
		}
	}

	// Verify the certificate chain
	if len(certs) > 0 {
		opts := x509.VerifyOptions{
			DNSName:       host,
			Intermediates: x509.NewCertPool(),
		}
		for _, cert := range certs[1:] {
			opts.Intermediates.AddCert(cert)
		}
		_, err := certs[0].Verify(opts)
		if err != nil {
			result.ChainValid = false
		}
	}

	return result
}

// ParsePEMCertificate parses a PEM-encoded certificate
func ParsePEMCertificate(pemData string) CertChainResult {
	result := CertChainResult{
		Certificates: make([]CertInfo, 0),
	}

	data := []byte(pemData)
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}

		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				result.Error = "Failed to parse certificate: " + err.Error()
				return result
			}
			result.Certificates = append(result.Certificates, parseCertificate(cert))
		}

		data = rest
	}

	if len(result.Certificates) == 0 {
		result.Error = "No valid certificates found in PEM data"
	}

	return result
}

func parseCertificate(cert *x509.Certificate) CertInfo {
	now := time.Now()
	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)

	ipAddrs := make([]string, 0, len(cert.IPAddresses))
	for _, ip := range cert.IPAddresses {
		ipAddrs = append(ipAddrs, ip.String())
	}

	sha1Sum := sha1.Sum(cert.Raw)
	sha256Sum := sha256.Sum256(cert.Raw)

	return CertInfo{
		Subject:            cert.Subject.String(),
		Issuer:             cert.Issuer.String(),
		SerialNumber:       cert.SerialNumber.String(),
		NotBefore:          cert.NotBefore,
		NotAfter:           cert.NotAfter,
		DaysUntilExpiry:    daysUntilExpiry,
		IsExpired:          now.After(cert.NotAfter),
		IsCA:               cert.IsCA,
		DNSNames:           cert.DNSNames,
		IPAddresses:        ipAddrs,
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
		PublicKeyAlgorithm: cert.PublicKeyAlgorithm.String(),
		Version:            cert.Version,
		FingerprintSHA1:    formatFingerprint(hex.EncodeToString(sha1Sum[:])),
		FingerprintSHA256:  formatFingerprint(hex.EncodeToString(sha256Sum[:])),
	}
}

func formatFingerprint(fp string) string {
	fp = strings.ToUpper(fp)
	var parts []string
	for i := 0; i < len(fp); i += 2 {
		end := i + 2
		if end > len(fp) {
			end = len(fp)
		}
		parts = append(parts, fp[i:end])
	}
	return strings.Join(parts, ":")
}

func getTLSVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}
