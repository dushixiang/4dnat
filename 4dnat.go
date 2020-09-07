package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const RetryInterval int = 5

func main() {
	printBanner()
	args := os.Args

	if len(args) < 4 {
		printHelp()
		os.Exit(-1)
	}

	switch args[1] {
	case "-listen":
	case "-l":
		listener(args[2], args[3])
		break
	case "-forward":
	case "-f":
		forward(args[2], args[3])
		break
	case "-agent":
	case "-a":
		agent(args[2], args[3])
		break
	default:
		printHelp()
	}
}

func printHelp() {
	println("usage:")
	println(`    "-forward listenPort targetAddress" example: "-forward 10000 127.0.0.1:22"`)
	println(`    "-listen listenPort0 listenPort1" example: "-listen 10000 10001"`)
	println(`    "-agent targetAddress0 targetAddress1" example: "-agent 127.0.0.1:10000 127.0.0.1:22"`)
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
	fmt.Printf("[#] [%s]->[%s] ==> [%s]->[%s]\n", src.RemoteAddr().String(), src.LocalAddr().String(), dest.LocalAddr().String(), dest.RemoteAddr().String())
	_, _ = io.Copy(src, dest)
	fmt.Printf("[-] [%s]->[%s] closed.\n", src.RemoteAddr().String(), src.LocalAddr().String())
	wg.Done()
}

func listener(listenPort0, listenPort1 string) {
	ln0 := listen(listenPort0)
	ln1 := listen(listenPort1)
	fmt.Printf("[#] 4dnat listen port on: [%s %s],[%s %s]\n", ln0.Addr().Network(), ln0.Addr().String(), ln1.Addr().Network(), ln1.Addr().String())

	for true {
		cc := make(chan net.Conn, 2)

		go accept(cc, ln0)
		go accept(cc, ln1)

		conn0, conn1 := <-cc, <-cc
		mutualCopyIO(conn0, conn1)
	}
}

func forward(listenPort string, targetAddress string) {
	ln := listen(listenPort)
	fmt.Printf("[#] 4dnat listen on: [%s %s] forward to: [%s]\n", ln.Addr().Network(), ln.Addr().String(), targetAddress)
	for true {
		cc := make(chan net.Conn, 2)

		go accept(cc, ln)
		go dial(cc, targetAddress)

		conn0, conn1 := <-cc, <-cc
		go mutualCopyIO(conn0, conn1)
	}
}

func agent(targetAddress0 string, targetAddress1 string) {
	fmt.Printf("[#] 4dnat agent with: [%s] [%s]\n", targetAddress0, targetAddress1)
	for {
		cc := make(chan net.Conn, 2)

		go dial(cc, targetAddress0)
		go dial(cc, targetAddress1)

		conn0, conn1 := <-cc, <-cc
		mutualCopyIO(conn0, conn1)
	}
}

func mutualCopyIO(conn0, conn1 net.Conn) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go copyIO(conn0, conn1, &wg)
	go copyIO(conn1, conn0, &wg)
	wg.Wait()
}

func dial(cc chan net.Conn, targetAddress string) {
	for true {
		conn, err := net.Dial("tcp", targetAddress)
		if err != nil {
			fmt.Printf("[x] connect [%s] error [%s].\n", targetAddress, err.Error())
			fmt.Printf("[#] retry to connect: [%s] after [%d] second\n", targetAddress, RetryInterval)
			time.Sleep(time.Duration(RetryInterval) * time.Second)
			continue
		}
		fmt.Printf("[+] [%s]->[%s] connected to target.\n", conn.LocalAddr().String(), targetAddress)
		cc <- conn
		break
	}
}

func accept(cc chan net.Conn, ln net.Listener) {
	for true {
		fmt.Printf("[#] 4dnat waiting for client to connect [%s %s]\n", ln.Addr().Network(), ln.Addr().String())
		c, err := ln.Accept()
		if err != nil {
			fmt.Printf("[x] accept error [%s].\n", err.Error())
			continue
		}
		fmt.Printf("[+] [%s]<-[%s] new client connected.\n", c.LocalAddr().String(), c.RemoteAddr().String())
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
		fmt.Printf("[x] listen error [%s].\n", err.Error())
		os.Exit(0)
	}
	return ln
}
