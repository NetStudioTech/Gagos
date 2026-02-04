package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type PingResult struct {
	Host        string    `json:"host"`
	IP          string    `json:"ip"`
	Success     bool      `json:"success"`
	PacketsSent int       `json:"packets_sent"`
	PacketsRecv int       `json:"packets_recv"`
	PacketLoss  float64   `json:"packet_loss"`
	MinRTT      float64   `json:"min_rtt_ms"`
	AvgRTT      float64   `json:"avg_rtt_ms"`
	MaxRTT      float64   `json:"max_rtt_ms"`
	RTTs        []float64 `json:"rtts,omitempty"`
	Error       string    `json:"error,omitempty"`
}

func Ping(host string, count int, timeout time.Duration) PingResult {
	result := PingResult{
		Host:        host,
		PacketsSent: count,
		RTTs:        make([]float64, 0),
	}

	// Resolve hostname to IP
	ips, err := net.LookupIP(host)
	if err != nil {
		result.Error = fmt.Sprintf("DNS resolution failed: %v", err)
		return result
	}

	if len(ips) == 0 {
		result.Error = "no IP addresses found"
		return result
	}

	// Prefer IPv4
	var targetIP net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			targetIP = ip
			break
		}
	}
	if targetIP == nil {
		targetIP = ips[0]
	}
	result.IP = targetIP.String()

	// Try ICMP first, fall back to TCP connectivity check
	icmpSuccess := false

	// Determine if IPv4 or IPv6
	isIPv4 := targetIP.To4() != nil
	var network string
	var proto int

	if isIPv4 {
		network = "ip4:icmp"
		proto = 1
	} else {
		network = "ip6:ipv6-icmp"
		proto = 58
	}

	// Try privileged ICMP
	conn, err := icmp.ListenPacket(network, "")
	if err == nil {
		defer conn.Close()
		icmpSuccess = true

		var minRTT, maxRTT, totalRTT float64
		minRTT = float64(timeout.Milliseconds())

		for i := 0; i < count; i++ {
			rtt, err := sendPing(conn, targetIP, proto, i, timeout, isIPv4)
			if err == nil {
				result.PacketsRecv++
				rttMs := float64(rtt.Microseconds()) / 1000.0
				result.RTTs = append(result.RTTs, rttMs)
				totalRTT += rttMs

				if rttMs < minRTT {
					minRTT = rttMs
				}
				if rttMs > maxRTT {
					maxRTT = rttMs
				}
			}

			if i < count-1 {
				time.Sleep(100 * time.Millisecond)
			}
		}

		if result.PacketsRecv > 0 {
			result.Success = true
			result.MinRTT = minRTT
			result.MaxRTT = maxRTT
			result.AvgRTT = totalRTT / float64(result.PacketsRecv)
		}
		result.PacketLoss = float64(result.PacketsSent-result.PacketsRecv) / float64(result.PacketsSent) * 100
	}

	// If ICMP failed, use TCP connectivity check (port 80 or 443)
	if !icmpSuccess {
		var minRTT, maxRTT, totalRTT float64
		minRTT = float64(timeout.Milliseconds())

		ports := []int{443, 80, 22}

		for i := 0; i < count; i++ {
			for _, port := range ports {
				start := time.Now()
				conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetIP.String(), port), timeout/time.Duration(count))
				rtt := time.Since(start)

				if err == nil {
					conn.Close()
					result.PacketsRecv++
					rttMs := float64(rtt.Microseconds()) / 1000.0
					result.RTTs = append(result.RTTs, rttMs)
					totalRTT += rttMs

					if rttMs < minRTT {
						minRTT = rttMs
					}
					if rttMs > maxRTT {
						maxRTT = rttMs
					}
					break // Success on this port, move to next ping
				}
			}

			if i < count-1 {
				time.Sleep(100 * time.Millisecond)
			}
		}

		if result.PacketsRecv > 0 {
			result.Success = true
			result.MinRTT = minRTT
			result.MaxRTT = maxRTT
			result.AvgRTT = totalRTT / float64(result.PacketsRecv)
		}
		result.PacketLoss = float64(result.PacketsSent-result.PacketsRecv) / float64(result.PacketsSent) * 100
	}

	return result
}

func sendPing(conn *icmp.PacketConn, target net.IP, proto int, seq int, timeout time.Duration, isIPv4 bool) (time.Duration, error) {
	var msg icmp.Message

	if isIPv4 {
		msg = icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   1,
				Seq:  seq,
				Data: []byte("GAGOS-PING"),
			},
		}
	} else {
		msg = icmp.Message{
			Type: ipv6.ICMPTypeEchoRequest,
			Code: 0,
			Body: &icmp.Echo{
				ID:   1,
				Seq:  seq,
				Data: []byte("GAGOS-PING"),
			},
		}
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return 0, err
	}

	start := time.Now()

	dst := &net.IPAddr{IP: target}

	if _, err := conn.WriteTo(msgBytes, dst); err != nil {
		return 0, err
	}

	conn.SetReadDeadline(time.Now().Add(timeout))
	reply := make([]byte, 1500)
	n, _, err := conn.ReadFrom(reply)
	if err != nil {
		return 0, err
	}

	rtt := time.Since(start)

	parsedMsg, err := icmp.ParseMessage(proto, reply[:n])
	if err != nil {
		return 0, err
	}

	switch parsedMsg.Type {
	case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
		return rtt, nil
	default:
		return 0, fmt.Errorf("unexpected ICMP type: %v", parsedMsg.Type)
	}
}

// DNS Lookup - pure Go

type DNSResult struct {
	Host       string   `json:"host"`
	Addresses  []string `json:"addresses,omitempty"`
	CNAME      string   `json:"cname,omitempty"`
	MX         []string `json:"mx,omitempty"`
	NS         []string `json:"ns,omitempty"`
	TXT        []string `json:"txt,omitempty"`
	PTR        []string `json:"ptr,omitempty"`
	RecordType string   `json:"record_type"`
	Error      string   `json:"error,omitempty"`
	Duration   float64  `json:"duration_ms"`
}

func DNSLookup(host string, recordType string) DNSResult {
	start := time.Now()
	result := DNSResult{
		Host:       host,
		RecordType: strings.ToUpper(recordType),
	}

	if result.RecordType == "" {
		result.RecordType = "A"
	}

	switch result.RecordType {
	case "A", "AAAA":
		ips, err := net.LookupIP(host)
		if err != nil {
			result.Error = err.Error()
		} else {
			for _, ip := range ips {
				if result.RecordType == "A" && ip.To4() != nil {
					result.Addresses = append(result.Addresses, ip.String())
				} else if result.RecordType == "AAAA" && ip.To4() == nil {
					result.Addresses = append(result.Addresses, ip.String())
				} else if result.RecordType == "A" {
					result.Addresses = append(result.Addresses, ip.String())
				}
			}
		}

	case "CNAME":
		cname, err := net.LookupCNAME(host)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.CNAME = cname
		}

	case "MX":
		mxs, err := net.LookupMX(host)
		if err != nil {
			result.Error = err.Error()
		} else {
			for _, mx := range mxs {
				result.MX = append(result.MX, fmt.Sprintf("%s (priority: %d)", mx.Host, mx.Pref))
			}
		}

	case "NS":
		nss, err := net.LookupNS(host)
		if err != nil {
			result.Error = err.Error()
		} else {
			for _, ns := range nss {
				result.NS = append(result.NS, ns.Host)
			}
		}

	case "TXT":
		txts, err := net.LookupTXT(host)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.TXT = txts
		}

	case "PTR":
		names, err := net.LookupAddr(host)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.PTR = names
		}

	default:
		result.Error = fmt.Sprintf("unsupported record type: %s (supported: A, AAAA, CNAME, MX, NS, TXT, PTR)", recordType)
	}

	result.Duration = float64(time.Since(start).Microseconds()) / 1000.0
	return result
}

// Port Check - pure Go

type PortCheckResult struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Open     bool   `json:"open"`
	Protocol string `json:"protocol"`
	Duration float64 `json:"duration_ms"`
	Error    string `json:"error,omitempty"`
	Banner   string `json:"banner,omitempty"`
}

func CheckPort(host string, port int, timeout time.Duration) PortCheckResult {
	start := time.Now()
	result := PortCheckResult{
		Host:     host,
		Port:     port,
		Protocol: "tcp",
	}

	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	result.Duration = float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		result.Open = false
		result.Error = err.Error()
		return result
	}
	defer conn.Close()

	result.Open = true

	// Try to grab banner (with short timeout)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	banner := make([]byte, 256)
	n, err := conn.Read(banner)
	if err == nil && n > 0 {
		result.Banner = strings.TrimSpace(string(banner[:n]))
	}

	return result
}

// Multi-port scan

type PortScanResult struct {
	Host   string            `json:"host"`
	Ports  []PortCheckResult `json:"ports"`
	Open   []int             `json:"open_ports"`
	Closed []int             `json:"closed_ports"`
	Total  int               `json:"total_scanned"`
	Error  string            `json:"error,omitempty"`
}

func ScanPorts(host string, ports []int, timeout time.Duration, concurrency int) PortScanResult {
	result := PortScanResult{
		Host:   host,
		Total:  len(ports),
		Open:   make([]int, 0),
		Closed: make([]int, 0),
	}

	if concurrency <= 0 {
		concurrency = 10
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, concurrency)

	for _, port := range ports {
		wg.Add(1)
		sem <- struct{}{}

		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()

			portResult := CheckPort(host, p, timeout)

			mu.Lock()
			result.Ports = append(result.Ports, portResult)
			if portResult.Open {
				result.Open = append(result.Open, p)
			} else {
				result.Closed = append(result.Closed, p)
			}
			mu.Unlock()
		}(port)
	}

	wg.Wait()
	return result
}

// Traceroute - pure Go using TCP with TTL

type TracerouteHop struct {
	Hop     int     `json:"hop"`
	Address string  `json:"address"`
	Host    string  `json:"host,omitempty"`
	RTT     float64 `json:"rtt_ms"`
	Error   string  `json:"error,omitempty"`
}

type TracerouteResult struct {
	Host      string          `json:"host"`
	TargetIP  string          `json:"target_ip"`
	Hops      []TracerouteHop `json:"hops"`
	Reached   bool            `json:"reached"`
	TotalHops int             `json:"total_hops"`
	Error     string          `json:"error,omitempty"`
}

func Traceroute(host string, maxHops int, timeout time.Duration) TracerouteResult {
	result := TracerouteResult{
		Host: host,
		Hops: make([]TracerouteHop, 0),
	}

	// Resolve target first
	ips, err := net.LookupIP(host)
	if err != nil {
		result.Error = fmt.Sprintf("DNS resolution failed: %v", err)
		return result
	}

	var targetIP net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			targetIP = ip
			break
		}
	}
	if targetIP == nil && len(ips) > 0 {
		targetIP = ips[0]
	}
	if targetIP == nil {
		result.Error = "no IP address found"
		return result
	}
	result.TargetIP = targetIP.String()

	// Try tracepath first (doesn't need raw sockets), fallback to traceroute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Try tracepath first - it doesn't require raw socket permissions
	cmd := exec.CommandContext(ctx, "tracepath", "-n", "-m", fmt.Sprintf("%d", maxHops), host)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	useTracepath := true

	if err != nil && len(outputStr) == 0 {
		// tracepath failed, try traceroute
		useTracepath = false
		ctx2, cancel2 := context.WithTimeout(context.Background(), timeout)
		defer cancel2()
		cmd = exec.CommandContext(ctx2, "traceroute", "-n", "-m", fmt.Sprintf("%d", maxHops), "-w", "2", host)
		output, err = cmd.CombinedOutput()
		outputStr = string(output)
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = "trace timed out"
			return result
		}
		// Check for permission errors
		if strings.Contains(outputStr, "Operation not permitted") || strings.Contains(outputStr, "Permission denied") {
			result.Error = "Trace requires permissions not available. Try ping or port-check instead."
			return result
		}
		// If error but some output, continue parsing
		if len(outputStr) == 0 {
			result.Error = fmt.Sprintf("trace failed: %v", err)
			return result
		}
	}

	// Parse output (both tracepath and traceroute have similar format)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "traceroute") || strings.HasPrefix(line, "tracepath") {
			continue
		}

		var hop TracerouteHop
		if useTracepath {
			hop = parseTracepathLine(line)
		} else {
			hop = parseTracerouteLine(line)
		}
		if hop.Hop > 0 {
			result.Hops = append(result.Hops, hop)
			result.TotalHops = hop.Hop
			if hop.Address == result.TargetIP {
				result.Reached = true
			}
		}
	}

	if len(result.Hops) > 0 && result.Hops[len(result.Hops)-1].Address == result.TargetIP {
		result.Reached = true
	}

	return result
}

// parseTracepathLine parses a line from tracepath output
// Format: " 1?: [LOCALHOST]     pmtu 1500"
// Or:     " 1:  192.168.1.1                                          0.123ms"
func parseTracepathLine(line string) TracerouteHop {
	hop := TracerouteHop{}
	line = strings.TrimSpace(line)

	// Skip pmtu lines and resume lines
	if strings.Contains(line, "pmtu") || strings.Contains(line, "Resume:") || strings.Contains(line, "Too many hops") {
		return hop
	}

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return hop
	}

	// First field is hop number with colon (e.g., "1:" or "1?:")
	hopStr := strings.TrimRight(fields[0], ":?")
	hopNum, err := strconv.Atoi(hopStr)
	if err != nil {
		return hop
	}
	hop.Hop = hopNum

	// Second field is IP or "no reply"
	if fields[1] == "no" {
		hop.Address = "*"
		return hop
	}

	// Check if it's an IP address or hostname
	addr := fields[1]
	if net.ParseIP(addr) != nil {
		hop.Address = addr
		// Try reverse DNS
		names, err := net.LookupAddr(addr)
		if err == nil && len(names) > 0 {
			hop.Host = strings.TrimSuffix(names[0], ".")
		}
	} else if addr == "[LOCALHOST]" {
		hop.Address = "127.0.0.1"
		hop.Host = "localhost"
	} else {
		hop.Address = addr
	}

	// Look for RTT (ends with "ms")
	for _, field := range fields[2:] {
		if strings.HasSuffix(field, "ms") {
			rttStr := strings.TrimSuffix(field, "ms")
			if rtt, err := strconv.ParseFloat(rttStr, 64); err == nil {
				hop.RTT = rtt
			}
			break
		}
	}

	return hop
}

func parseTracerouteLine(line string) TracerouteHop {
	hop := TracerouteHop{}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return hop
	}

	// First field is hop number
	hopNum, err := strconv.Atoi(fields[0])
	if err != nil {
		return hop
	}
	hop.Hop = hopNum

	// Parse remaining fields for IP and RTT
	for i := 1; i < len(fields); i++ {
		field := fields[i]
		if field == "*" {
			if hop.Address == "" {
				hop.Address = "*"
			}
			continue
		}
		// Check if it's an IP address
		if net.ParseIP(field) != nil {
			hop.Address = field
			// Try reverse DNS
			names, err := net.LookupAddr(field)
			if err == nil && len(names) > 0 {
				hop.Host = strings.TrimSuffix(names[0], ".")
			}
		}
		// Check if it's RTT (ends with "ms")
		if strings.HasSuffix(field, "ms") {
			rttStr := strings.TrimSuffix(field, "ms")
			if rtt, err := strconv.ParseFloat(rttStr, 64); err == nil {
				hop.RTT = rtt
			}
		} else if i+1 < len(fields) && fields[i+1] == "ms" {
			if rtt, err := strconv.ParseFloat(field, 64); err == nil {
				hop.RTT = rtt
			}
		}
	}

	if hop.Address == "" {
		hop.Address = "*"
	}

	return hop
}

// HTTP Check - pure Go

type HTTPCheckResult struct {
	URL          string            `json:"url"`
	StatusCode   int               `json:"status_code"`
	Status       string            `json:"status"`
	ResponseTime float64           `json:"response_time_ms"`
	Headers      map[string]string `json:"headers,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	Size         int64             `json:"size_bytes"`
	Error        string            `json:"error,omitempty"`
}

func HTTPCheck(url string, timeout time.Duration, followRedirects bool) HTTPCheckResult {
	result := HTTPCheckResult{
		URL:     url,
		Headers: make(map[string]string),
	}

	client := &http.Client{
		Timeout: timeout,
	}

	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	start := time.Now()
	resp, err := client.Get(url)
	result.ResponseTime = float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Status = resp.Status
	result.ContentType = resp.Header.Get("Content-Type")
	result.Size = resp.ContentLength

	for _, header := range []string{"Server", "X-Powered-By", "Content-Encoding"} {
		if val := resp.Header.Get(header); val != "" {
			result.Headers[header] = val
		}
	}

	return result
}
