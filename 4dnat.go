package main

import (
	"fmt"
	"io"
	"net"
	"os"
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

	mode := args[1]

	switch mode {
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
	defer dest.Close()
	_, _ = io.Copy(src, dest)
	fmt.Printf("[-] [%s]--[%s] disconnected.\n", src.LocalAddr().String(), src.RemoteAddr().String())
	fmt.Printf("[-] [%s]--[%s] disconnected.\n", dest.LocalAddr().String(), dest.RemoteAddr().String())
	wg.Done()
}

// 监听端口模式
func listener(listenPort0, listenPort1 string) {
	ln0 := listen(listenPort0)
	ln1 := listen(listenPort1)
	fmt.Printf("[#] 4dnat listen on: [%s] [%s]\n", listenPort0, listenPort1)

	for {
		conn0, err0 := accept(ln0)
		if err0 != nil {
			continue
		}
		conn1, err1 := accept(ln1)
		if err1 != nil {
			continue
		}

		go func() {
			establishChannel(conn0, conn1)
		}()
	}

}

// 端口转发模式
func forward(listenPort string, targetAddress string) {
	ln := listen(listenPort)
	fmt.Printf("[#] 4dnat listen on: [%s] forward to: [%s]\n", listenPort, targetAddress)
	for {
		conn0, err := accept(ln)
		if err != nil {
			time.Sleep(time.Duration(RetryInterval) * time.Second)
			continue
		}
		go func() {
			// after server accept will be connect the target address,if failed will be retry.
			for true {
				conn1, err := dial(targetAddress)
				if err != nil {
					time.Sleep(time.Duration(RetryInterval) * time.Second)
					continue
				}

				go func() {
					establishChannel(conn0, conn1)
				}()
				break
			}
		}()
	}
}

// 通道模式
func agent(targetAddress0 string, targetAddress1 string) {

	fmt.Printf("[#] 4dnat agent with: [%s] [%s]\n", targetAddress0, targetAddress1)

	for {
		conn0, err0 := dial(targetAddress0)
		if err0 != nil {
			time.Sleep(time.Duration(RetryInterval) * time.Second)
			continue
		}
		conn1, err1 := dial(targetAddress1)
		if err1 != nil {
			if conn0 != nil {
				conn0.Close()
				fmt.Printf("[-] [%s]--[%s] active disconnect.\n", conn0.LocalAddr().String(), conn0.RemoteAddr().String())
			}
			time.Sleep(time.Duration(RetryInterval) * time.Second)
			continue
		}

		establishChannel(conn0, conn1)
	}

}

func establishChannel(conn0 net.Conn, conn1 net.Conn) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go copyIO(conn0, conn1, &wg)
	go copyIO(conn1, conn0, &wg)
	wg.Wait()
}

func dial(targetAddress string) (net.Conn, error) {
	conn, err := net.Dial("tcp", targetAddress)
	if err != nil {
		fmt.Printf("[x] connected [%s] error [%s].\n", targetAddress, err.Error())
		return conn, err
	}
	fmt.Printf("[+] [%s]->[%s] connected to target.\n", conn.LocalAddr().String(), targetAddress)
	return conn, err
}

func accept(ln net.Listener) (net.Conn, error) {
	conn, err := ln.Accept()
	if err != nil {
		fmt.Printf("[x] accept error [%s].\n", err.Error())
		return nil, err
	}
	fmt.Printf("[+] [%s]<-[%s] new client connected.\n", conn.LocalAddr().String(), conn.RemoteAddr().String())
	return conn, nil
}

func listen(listenPort string) net.Listener {
	ln, err := net.Listen("tcp", ":"+listenPort)
	if err != nil {
		panic(err)
	}
	return ln
}
