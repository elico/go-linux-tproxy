package main

import (
	"bufio"
	"net"
	"os"
	"strings"
	"unicode"

	"github.com/asaskevich/govalidator"
)

// RandomHosts ---
var RandomHosts []string

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Hosts ---
func Hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// remove network address and broadcast address
	return ips[1 : len(ips)-1], nil
}

const commentChars = "#;"

func stripComment(source string) string {
	if cut := strings.IndexAny(source, commentChars); cut >= 0 {
		return strings.TrimRightFunc(source[:cut], unicode.IsSpace)
	}
	return source
}

func processCidrLine(line string) string {
	line = stripComment(strings.TrimSpace(line))
	if govalidator.IsCIDR(line) {
		return line
	}
	return ""

}

// ReadCidrListFile ---
func ReadCidrListFile(filename string) {
	randomHosts := []string{}
	file, err := os.Open(filename)
	if err != nil {
		debugerr(err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := processCidrLine(scanner.Text())
		if line != "" {
			hosts, err := Hosts(line)
			debugerr(err)
			if err == nil {
				for _, host := range hosts {
					randomHosts = append(randomHosts, host)
				}
			}
		}

	}

	if err := scanner.Err(); err != nil {
		debugerr(err)
	}
	if len(randomHosts) > 0 {
		RandomHosts = randomHosts
	}
}
