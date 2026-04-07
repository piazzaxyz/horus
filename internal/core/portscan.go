package core

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

// CommonPorts maps well-known port numbers to their service names.
var CommonPorts = map[int]string{
	21:    "FTP",
	22:    "SSH",
	23:    "Telnet",
	25:    "SMTP",
	53:    "DNS",
	80:    "HTTP",
	110:   "POP3",
	111:   "RPC",
	135:   "MSRPC",
	139:   "NetBIOS",
	143:   "IMAP",
	443:   "HTTPS",
	445:   "SMB",
	465:   "SMTPS",
	587:   "SMTP-Submission",
	993:   "IMAPS",
	995:   "POP3S",
	1433:  "MSSQL",
	1521:  "Oracle",
	3306:  "MySQL",
	3389:  "RDP",
	5432:  "PostgreSQL",
	5900:  "VNC",
	6379:  "Redis",
	8080:  "HTTP-Alt",
	8443:  "HTTPS-Alt",
	8888:  "HTTP-Dev",
	9200:  "Elasticsearch",
	9300:  "Elasticsearch-Transport",
	27017: "MongoDB",
	27018: "MongoDB-Shard",
	5672:  "AMQP",
	15672: "RabbitMQ-Mgmt",
	2181:  "Zookeeper",
	9092:  "Kafka",
	2379:  "etcd",
	2380:  "etcd-Peer",
	4200:  "CockroachDB-UI",
	5984:  "CouchDB",
	7474:  "Neo4j",
	8086:  "InfluxDB",
	11211: "Memcached",
	4567:  "Sinatra",
	4000:  "HTTP-Dev-Alt",
	3000:  "NodeJS-Dev",
	5000:  "Flask-Dev",
	8000:  "HTTP-Dev2",
}

// ScanPort attempts to connect to a single port and returns the result.
func ScanPort(host string, port int, timeout time.Duration) PortResult {
	result := PortResult{
		Port: port,
	}

	if service, ok := CommonPorts[port]; ok {
		result.Service = service
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	result.Duration = time.Since(start)

	if err != nil {
		result.Open = false
		return result
	}
	defer conn.Close()

	result.Open = true

	// Try to read a banner
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	if n > 0 {
		banner := string(buf[:n])
		// Clean up banner
		cleaned := ""
		for _, ch := range banner {
			if ch >= 32 && ch < 127 {
				cleaned += string(ch)
			} else if ch == '\n' || ch == '\r' {
				cleaned += " "
			}
		}
		if len(cleaned) > 100 {
			cleaned = cleaned[:100] + "..."
		}
		result.Banner = cleaned
	}

	return result
}

// ScanPorts scans a range of ports on the given host concurrently.
// Returns only open ports, sorted by port number.
func ScanPorts(host string, startPort, endPort int, concurrency int) []PortResult {
	if concurrency <= 0 {
		concurrency = 100
	}

	portCount := endPort - startPort + 1
	if portCount <= 0 {
		return nil
	}

	jobs := make(chan int, portCount)
	for p := startPort; p <= endPort; p++ {
		jobs <- p
	}
	close(jobs)

	resultsCh := make(chan PortResult, portCount)
	var wg sync.WaitGroup

	timeout := 1 * time.Second

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				r := ScanPort(host, port, timeout)
				if r.Open {
					resultsCh <- r
				}
			}
		}()
	}

	wg.Wait()
	close(resultsCh)

	var results []PortResult
	for r := range resultsCh {
		results = append(results, r)
	}

	// Sort by port number
	sort.Slice(results, func(i, j int) bool {
		return results[i].Port < results[j].Port
	})

	return results
}
