# 4DNAT

English | [简体中文](./README-zh_CN.md)

## Introduction

The 4DNAT was named from 4 and DNAT. This tool works in the fourth layer of transport layer of the OSI model, while 4 and for sound, means a tool that is dedicated to the target address conversion. 4dnat develops using Go language, has natural cross-platform, and uses the GO standard library development, without any third-party dependence, only one binary executable after compiling. It has four working modes:

### Forward

Accept two parameters, listen port, and destination addresses, actively connect the target address after receiving the request in the listening port, example:

```bash
./4dnat -forward 2222 192.168.1.100:22
```

### Listen

Accept two parameters, listen port 1 and monitor port 2, and exchange data received by two ports, example:

```bash
./4dnat -listen 10000 10001
```

### Agent

Accept two parameters, target addresses 1, and destination address 2, actively connect these two target addresses after startup, and exchange data received by the two ports, example:

```bash
./4dnat -agent 127.0.0.1:10000 127.0.0.1:22
```

### http/https proxy

Accept two parameters or four parameters, proxy types, listener ports, certificate paths, and private key paths, examples:

#### http proxy

```shell script
./4dnat -proxy http 1080
```

#### https proxy

```shell script
./4dnat -proxy https 1080 server.crt server.key
```