// Copyright 2024-2026 GAGOS Project
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Telnet - TCP connection with send/receive
type TelnetResult struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Connected bool   `json:"connected"`
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
	Duration float64 `json:"duration_ms"`
}

func TelnetConnect(host string, port int, command string, timeout time.Duration) TelnetResult {
	start := time.Now()
	result := TelnetResult{
		Host: host,
		Port: port,
	}

	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", err)
		result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
		return result
	}
	defer conn.Close()

	result.Connected = true

	// Set read/write deadline
	conn.SetDeadline(time.Now().Add(timeout))

	var response strings.Builder

	// Read any banner first (with short timeout)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	banner := make([]byte, 4096)
	n, err := conn.Read(banner)
	if err == nil && n > 0 {
		response.WriteString(string(banner[:n]))
	}

	// Send command if provided
	if command != "" {
		conn.SetWriteDeadline(time.Now().Add(timeout))
		if !strings.HasSuffix(command, "\n") {
			command += "\r\n"
		}
		_, err = conn.Write([]byte(command))
		if err != nil {
			result.Error = fmt.Sprintf("Write failed: %v", err)
			result.Response = response.String()
			result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
			return result
		}

		// Read response
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 8192)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				response.WriteString(string(buf[:n]))
			}
			if err != nil {
				break
			}
			if response.Len() > 65536 { // Max 64KB response
				break
			}
		}
	}

	result.Response = response.String()
	result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
	return result
}

// Whois lookup
type WhoisResult struct {
	Query    string `json:"query"`
	Response string `json:"response"`
	Server   string `json:"server"`
	Error    string `json:"error,omitempty"`
	Duration float64 `json:"duration_ms"`
}

func Whois(query string, timeout time.Duration) WhoisResult {
	start := time.Now()
	result := WhoisResult{
		Query: query,
	}

	// Determine whois server based on query type
	server := "whois.iana.org"

	// For domains, try to find the right whois server
	if strings.Contains(query, ".") {
		parts := strings.Split(query, ".")
		tld := parts[len(parts)-1]

		// Common TLD whois servers
		whoisServers := map[string]string{
			"com":  "whois.verisign-grs.com",
			"net":  "whois.verisign-grs.com",
			"org":  "whois.pir.org",
			"info": "whois.afilias.net",
			"io":   "whois.nic.io",
			"co":   "whois.nic.co",
			"me":   "whois.nic.me",
			"de":   "whois.denic.de",
			"uk":   "whois.nic.uk",
			"fr":   "whois.nic.fr",
			"eu":   "whois.eu",
			"ru":   "whois.tcinet.ru",
			"ge":   "whois.nic.ge",
		}

		if srv, ok := whoisServers[strings.ToLower(tld)]; ok {
			server = srv
		}
	}

	// For IP addresses, use ARIN/RIPE
	if net.ParseIP(query) != nil {
		server = "whois.arin.net"
	}

	result.Server = server

	conn, err := net.DialTimeout("tcp", server+":43", timeout)
	if err != nil {
		result.Error = fmt.Sprintf("Connection to %s failed: %v", server, err)
		result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
		return result
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	// Send query
	_, err = conn.Write([]byte(query + "\r\n"))
	if err != nil {
		result.Error = fmt.Sprintf("Write failed: %v", err)
		result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
		return result
	}

	// Read response
	var response strings.Builder
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				// Ignore read errors at end
			}
			break
		}
		response.WriteString(line)
		if response.Len() > 65536 { // Max 64KB
			break
		}
	}

	result.Response = response.String()
	result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
	return result
}

// SSL Certificate checker
type SSLCertResult struct {
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Valid       bool      `json:"valid"`
	Subject     string    `json:"subject,omitempty"`
	Issuer      string    `json:"issuer,omitempty"`
	NotBefore   string    `json:"not_before,omitempty"`
	NotAfter    string    `json:"not_after,omitempty"`
	DaysLeft    int       `json:"days_left,omitempty"`
	DNSNames    []string  `json:"dns_names,omitempty"`
	Version     int       `json:"version,omitempty"`
	SerialNumber string   `json:"serial_number,omitempty"`
	Error       string    `json:"error,omitempty"`
	Duration    float64   `json:"duration_ms"`
}

func CheckSSL(host string, port int, timeout time.Duration) SSLCertResult {
	start := time.Now()
	result := SSLCertResult{
		Host: host,
		Port: port,
	}

	if port == 0 {
		port = 443
		result.Port = port
	}

	address := fmt.Sprintf("%s:%d", host, port)

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", address, &tls.Config{
		InsecureSkipVerify: true, // We want to check the cert even if invalid
	})
	if err != nil {
		result.Error = fmt.Sprintf("TLS connection failed: %v", err)
		result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
		return result
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		result.Error = "No certificates found"
		result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
		return result
	}

	cert := certs[0]

	result.Subject = cert.Subject.CommonName
	result.Issuer = cert.Issuer.CommonName
	result.NotBefore = cert.NotBefore.Format(time.RFC3339)
	result.NotAfter = cert.NotAfter.Format(time.RFC3339)
	result.DNSNames = cert.DNSNames
	result.Version = cert.Version
	result.SerialNumber = cert.SerialNumber.String()

	// Calculate days left
	daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
	result.DaysLeft = daysLeft

	// Check validity
	now := time.Now()
	if now.After(cert.NotBefore) && now.Before(cert.NotAfter) {
		result.Valid = true
	}

	result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
	return result
}

// Enhanced HTTP/Curl client
type CurlResult struct {
	URL           string              `json:"url"`
	Method        string              `json:"method"`
	StatusCode    int                 `json:"status_code"`
	Status        string              `json:"status"`
	Headers       map[string][]string `json:"headers,omitempty"`
	Body          string              `json:"body,omitempty"`
	ContentType   string              `json:"content_type,omitempty"`
	ContentLength int64               `json:"content_length"`
	RedirectURL   string              `json:"redirect_url,omitempty"`
	TLSVersion    string              `json:"tls_version,omitempty"`
	Protocol      string              `json:"protocol,omitempty"`
	Duration      float64             `json:"duration_ms"`
	Error         string              `json:"error,omitempty"`
}

func Curl(url string, method string, headers map[string]string, body string, timeout time.Duration, followRedirects bool, includeBody bool) CurlResult {
	start := time.Now()
	result := CurlResult{
		URL:     url,
		Method:  method,
		Headers: make(map[string][]string),
	}

	if method == "" {
		method = "GET"
		result.Method = method
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		result.Error = fmt.Sprintf("Request creation failed: %v", err)
		result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
		return result
	}

	// Add custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Set User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "GAGOS/1.0 (Network Multi-Tool)")
	}

	resp, err := client.Do(req)
	result.Duration = float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		result.Error = fmt.Sprintf("Request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Status = resp.Status
	result.ContentType = resp.Header.Get("Content-Type")
	result.ContentLength = resp.ContentLength
	result.Protocol = resp.Proto

	// Copy response headers
	for k, v := range resp.Header {
		result.Headers[k] = v
	}

	// Get redirect URL if any
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		result.RedirectURL = resp.Header.Get("Location")
	}

	// TLS info
	if resp.TLS != nil {
		switch resp.TLS.Version {
		case tls.VersionTLS10:
			result.TLSVersion = "TLS 1.0"
		case tls.VersionTLS11:
			result.TLSVersion = "TLS 1.1"
		case tls.VersionTLS12:
			result.TLSVersion = "TLS 1.2"
		case tls.VersionTLS13:
			result.TLSVersion = "TLS 1.3"
		}
	}

	// Read body if requested (limit to 1MB)
	if includeBody {
		limitedReader := io.LimitReader(resp.Body, 1024*1024)
		bodyBytes, err := io.ReadAll(limitedReader)
		if err == nil {
			result.Body = string(bodyBytes)
		}
	}

	return result
}

// Network interface info
type InterfaceInfo struct {
	Name       string   `json:"name"`
	MTU        int      `json:"mtu"`
	HWAddr     string   `json:"hw_addr,omitempty"`
	Flags      string   `json:"flags"`
	Addresses  []string `json:"addresses"`
}

type NetworkInfoResult struct {
	Hostname   string          `json:"hostname,omitempty"`
	Interfaces []InterfaceInfo `json:"interfaces"`
	Error      string          `json:"error,omitempty"`
}

func GetNetworkInfo() NetworkInfoResult {
	result := NetworkInfoResult{
		Interfaces: make([]InterfaceInfo, 0),
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		result.Error = err.Error()
		return result
	}

	for _, iface := range interfaces {
		info := InterfaceInfo{
			Name:      iface.Name,
			MTU:       iface.MTU,
			HWAddr:    iface.HardwareAddr.String(),
			Flags:     iface.Flags.String(),
			Addresses: make([]string, 0),
		}

		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				info.Addresses = append(info.Addresses, addr.String())
			}
		}

		result.Interfaces = append(result.Interfaces, info)
	}

	return result
}
