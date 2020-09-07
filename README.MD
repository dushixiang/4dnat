## 4DNAT 
go 语言实现的数据转发工具

## 使用方式
1. 转发模式

    ```
    ./4dnat -forward listenPort targetAddress
    ```
    示例：监听本地10000端口，接收数据后转发到127.0.0.1的22端口
    ```
    ./4dnat -forward 10000 127.0.0.1:22
    ```
2. 监听模式

    ```
    ./4dnat -listen listenPort0 listenPort1
    ```
    示例：监听本地10000和10001端口，并交换两个端口接收到的数据
    ```
    ./4dnat -listen 10000 10001
    ```
3. 代理人模式

    ```
    ./4dnat -agent targetAddress0 targetAddress1
    ```
    示例：主动连接127.0.0.1的10000端口和127.0.0.1的22端口，并交换两个端口接收到的数据
    ```
    ./4dnat -agent 127.0.0.1:10000 127.0.0.1:22
    ```
   
## 其他
- -forward 简写为-f
- -listen 简写为-l
- -agent 简写为-a