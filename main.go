package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"
)

func scanPorts(ip string, ports []int, sem chan struct{}, file *os.File) {

	for _, port := range ports {
		sem <- struct{}{} // 占用一个通道空位，如果通道已满，这里会阻塞直到有空位
		go func(p int) {
			defer func() {
				<-sem
			}() // 释放通道空位

			target := fmt.Sprintf("%s:%d", ip, p)

			conn, err := net.DialTimeout("tcp", target, time.Second*1)
			if err != nil {
				return
			}
			defer conn.Close()

			output := fmt.Sprintf("Port %d on %s is open\n", p, ip)
			fmt.Print(output)                 // 将输出打印到控制台
			_, err = file.WriteString(output) // 将输出追加写入文件
			if err != nil {
				fmt.Println("Error writing to file:", err)
			}
		}(port)
	}
}

func main() {
	var ip string
	var portsStr string
	var maxThreads int

	flag.StringVar(&ip, "ip", "", "IP address or range to scan")
	flag.StringVar(&portsStr, "p", "80", "Ports to scan (comma-separated or range)")
	flag.IntVar(&maxThreads, "t", 100, "Number of threads to use")

	flag.Parse()

	if ip == "" {
		fmt.Println("Please provide an IP address to scan")
		return
	}

	var ports []int
	portRanges := strings.Split(portsStr, ",")
	for _, prange := range portRanges {
		if strings.Contains(prange, "-") {
			rangeBounds := strings.Split(prange, "-")
			startPort := atoi(rangeBounds[0])
			endPort := atoi(rangeBounds[1])
			for i := startPort; i <= endPort; i++ {
				ports = append(ports, i)
			}
		} else {
			ports = append(ports, atoi(prange))
		}
	}
	sem := make(chan struct{}, maxThreads) // 创建带缓冲通道

	ipList := expandIPRange(ip)
	file, err := os.OpenFile("output.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	for _, ip := range ipList {
		scanPorts(ip, ports, sem, file)
	}
	close(sem) // 关闭通道
}
func expandIPRange(ipRange string) []string {
	ipList := []string{}

	if strings.Contains(ipRange, "-") || net.ParseIP(ipRange) != nil {
		ipRangeSlice := strings.Split(ipRange, "-")
		if len(ipRangeSlice) < 2 {
			ipList = append(ipList, ipRange)
			return ipList
		}
		startIP := net.ParseIP(ipRangeSlice[0])
		endIP := net.ParseIP(ipRangeSlice[1])

		if startIP.To4() == nil || endIP.To4() == nil {
			// Not IPv4 addresses
			return ipList
		}

		for ip := startIP.To4(); bytes.Compare(ip, endIP.To4()) <= 0; incIP(ip) {
			ipList = append(ipList, ip.String())
		}
	} else {
		// Check if ipRange is a filename
		fileContent, err := ioutil.ReadFile(ipRange)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return ipList
		}

		fileIPs := strings.Split(string(fileContent), "\n")
		for _, ip := range fileIPs {
			ip = strings.TrimSpace(ip)
			if net.ParseIP(ip) != nil {
				ipList = append(ipList, ip)
			}
		}
	}

	return ipList
}

func atoi(s string) int {
	val := 0
	for i := 0; i < len(s); i++ {
		val = val*10 + int(s[i]-'0')
	}
	return val
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
