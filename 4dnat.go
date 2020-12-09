package main

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const RetryInterval int = 5
const Timeout int = 10
const Version = "v0.0.4"

func init() {
	log.SetPrefix("[4dnat] ")
}

func main() {
	printBanner()
	args := os.Args

	if len(args) == 2 {
		switch args[1] {
		case "-version", "-v", "-V":
			println(Version)
			os.Exit(0)
		}
	}

	if len(args) < 4 {
		printHelp()
		os.Exit(0)
	}

	switch args[1] {
	case "-listen", "-l":
		listener(args[2], args[3])
	case "-agent", "-a":
		agent(args[2], args[3])
	case "-forward", "-f":
		forward(args[2], args[3])
	case "-proxy", "-p":
		proxy(args[2], args[3], args)
	default:
		printHelp()
	}
}

func printHelp() {
	println("usage:")
	println(`    "-forward listenPort targetAddress" example: "-forward 10000 127.0.0.1:22"`)
	println(`    "-listen listenPort0 listenPort1" example: "-listen 10000 10001"`)
	println(`    "-agent targetAddress0 targetAddress1" example: "-agent 127.0.0.1:10000 127.0.0.1:22"`)
	println(`    "-proxy protocol listenAddress" example: "-proxy http 1080", "-proxy https 1080 server.pem server.key", "-proxy socks5 1080"`)
	println(`    "-version"`)
}

func printBanner() {
	println(`
   _____     .___             __   
  /  |  |  __| _/____ _____ _/  |_ 
 /   |  |_/ __ |/    \\__  \\   __\
/    ^   / /_/ |   |  \/ __ \|  |  
\____   |\____ |___|  (____  /__|  
     |__|     \/    \/     \/
`)
}

func copyIO(src, dest net.Conn, wg *sync.WaitGroup) {
	defer src.Close()
	_, _ = io.Copy(src, dest)
	wg.Done()
}

func mutualCopyIO(conn0, conn1 net.Conn) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	log.Printf("[*] [%s <-> %s] :) [%s <-> %s]\n", conn0.RemoteAddr().String(), conn0.LocalAddr().String(), conn1.LocalAddr().String(), conn1.RemoteAddr().String())
	go copyIO(conn0, conn1, &wg)
	go copyIO(conn1, conn0, &wg)
	log.Printf("[-] [%s <-> %s] :( [%s <-> %s]\n", conn0.RemoteAddr().String(), conn0.LocalAddr().String(), conn1.LocalAddr().String(), conn1.RemoteAddr().String())
	wg.Wait()
}

func listener(listenPort0, listenPort1 string) {
	ln0 := listen(listenPort0)
	ln1 := listen(listenPort1)
	log.Printf("[*] listen port on: [%s %s],[%s %s]\n", ln0.Addr().Network(), ln0.Addr().String(), ln1.Addr().Network(), ln1.Addr().String())

	for {
		cc := make(chan net.Conn, 2)

		go accept(cc, ln0)
		go accept(cc, ln1)

		conn0, conn1 := <-cc, <-cc
		go mutualCopyIO(conn0, conn1)
	}
}

func forward(listenPort string, targetAddress string) {
	ln := listen(listenPort)
	log.Printf("[*] listen on: [%s %s] forward to: [%s]\n", ln.Addr().Network(), ln.Addr().String(), targetAddress)
	for {
		log.Printf("[*] waiting for client to connect [%s %s]\n", ln.Addr().Network(), ln.Addr().String())
		conn0, err := ln.Accept()
		if err != nil {
			log.Printf("[x] accept error [%s]\n", err.Error())
			continue
		}
		log.Printf("[+] [%s <- %s] new client connected\n", conn0.LocalAddr().String(), conn0.RemoteAddr().String())

		go handleForward(targetAddress, conn0)
	}
}

func handleForward(targetAddress string, conn0 net.Conn) {
	conn1, err := net.DialTimeout("tcp", targetAddress, time.Duration(Timeout)*time.Second)
	if err != nil {
		log.Printf("[x] connect [%s] error [%s]\n", targetAddress, err.Error())
		_, _ = conn0.Write([]byte(err.Error()))
		return
	}

	mutualCopyIO(conn0, conn1)
}

func agent(targetAddress0 string, targetAddress1 string) {
	log.Printf("[*] agent with: [%s %s]\n", targetAddress0, targetAddress1)
	for {
		conn0, err := net.DialTimeout("tcp", targetAddress0, time.Duration(Timeout)*time.Second)
		if err != nil {
			log.Printf("[x] connect [%s] error [%s]\n", targetAddress0, err.Error())
			log.Printf("[*] retry to connect: [%s] after [%d] second\n", targetAddress0, RetryInterval)
			time.Sleep(time.Duration(RetryInterval) * time.Second)
			continue
		}
		log.Printf("[+] [%s <-> %s] connected to target\n", conn0.LocalAddr().String(), targetAddress0)

		var conn1 net.Conn
		for {
			conn1, err = net.DialTimeout("tcp", targetAddress1, time.Duration(Timeout)*time.Second)
			if err != nil {
				log.Printf("[x] connect [%s] error [%s]\n", targetAddress1, err.Error())
				log.Printf("[*] retry to connect: [%s] after [%d] second\n", targetAddress1, RetryInterval)
				time.Sleep(time.Duration(RetryInterval) * time.Second)
				continue
			}
			log.Printf("[+] [%s <-> %s] connected to target\n", conn1.LocalAddr().String(), targetAddress1)
			break
		}

		mutualCopyIO(conn0, conn1)
	}
}

func accept(cc chan net.Conn, ln net.Listener) {
	for {
		log.Printf("[*] waiting for client to connect [%s %s]\n", ln.Addr().Network(), ln.Addr().String())
		c, err := ln.Accept()
		if err != nil {
			log.Printf("[x] accept error [%s]\n", err.Error())
			continue
		}
		log.Printf("[+] [%s <- %s] new client connected\n", c.LocalAddr().String(), c.RemoteAddr().String())
		cc <- c
		break
	}
}

func listen(listenPort string) net.Listener {
	if !strings.Contains(listenPort, ":") {
		listenPort = "0.0.0.0:" + listenPort
	}
	ln, err := net.Listen("tcp", listenPort)
	if err != nil {
		log.Printf("[x] listen error [%s]\n", err.Error())
		os.Exit(0)
	}
	return ln
}

func proxy(protocol, listenAddress string, args []string) {
	if protocol != "http" && protocol != "https" && protocol != "socks5" {
		log.Fatal("protocol must be either http or https or socks5")
	}

	if !strings.Contains(listenAddress, ":") {
		listenAddress = "0.0.0.0:" + listenAddress
	}

	if "socks5" == protocol {
		ln := listen(listenAddress)
		for {
			log.Printf("[*] waiting for client to connect [%s %s]\n", ln.Addr().Network(), ln.Addr().String())
			conn0, err := ln.Accept()
			if err != nil {
				log.Printf("[x] accept error [%s]\n", err.Error())
				continue
			}
			log.Printf("[+] [%s <- %s] new client connected\n", conn0.LocalAddr().String(), conn0.RemoteAddr().String())

			go handleSocks5(conn0)
		}
	} else {
		server := &http.Server{
			Addr: listenAddress,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodConnect {
					handleTunneling(w, r)
				} else {
					handleHttp(w, r)
				}
			}),
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}

		if protocol == "https" {
			if len(args) < 6 {
				printHelp()
				os.Exit(0)
			}
			certFile := args[5]
			keyFile := args[6]
			log.Printf("[*] https proxy listen on: [%s] with cert: [%s] and key: [%s]", listenAddress, certFile, keyFile)
			log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
		} else {
			log.Printf("[*] http proxy listen on: [%s]", listenAddress)
			log.Fatal(server.ListenAndServe())
		}
	}
}

func parseTargetAddress(c net.Conn) (string, error) {
	var buf [1024]byte
	n, err := c.Read(buf[:])
	if err != nil {
		return "", err
	}

	if buf[0] == 0x05 {
		_, _ = c.Write([]byte{0x05, 0x00})
		n, err = c.Read(buf[:])
		if err != nil {
			return "", err
		}

		var host, port string
		switch buf[3] {
		case 0x01:
			host = net.IPv4(buf[4], buf[5], buf[6], buf[7]).String()
		case 0x03:
			host = string(buf[5 : n-2])
		case 0x04:
			host = net.IP{buf[4], buf[5], buf[6], buf[7], buf[8], buf[9], buf[10], buf[11], buf[12], buf[13], buf[14], buf[15], buf[16], buf[17], buf[18], buf[19]}.String()
		}
		port = strconv.Itoa(int(buf[n-2])<<8 | int(buf[n-1]))

		targetAddress := net.JoinHostPort(host, port)
		return targetAddress, nil
	}
	return "", errors.New("unsupported protocol")
}

func handleSocks5(conn0 net.Conn) {
	targetAddress, err := parseTargetAddress(conn0)
	if err != nil {
		log.Printf("[x] parse target address error [%s]\n", err.Error())
		_, _ = conn0.Write([]byte(err.Error()))
		return
	}
	conn1, err := net.DialTimeout("tcp", targetAddress, time.Duration(Timeout)*time.Second)
	if err != nil {
		log.Printf("[x] connect [%s] error [%s]\n", targetAddress, err.Error())
		_, _ = conn0.Write([]byte(err.Error()))
		return
	}
	log.Printf("[+] [%s -> %s] connected to target\n", conn1.LocalAddr().String(), targetAddress)

	_, _ = conn0.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	mutualCopyIO(conn0, conn1)
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	remoteConn, err := net.DialTimeout("tcp", r.Host, time.Duration(Timeout)*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}
	centralConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	go mutualCopyIO(remoteConn, centralConn)
}

func handleHttp(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
