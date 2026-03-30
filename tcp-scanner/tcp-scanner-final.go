/*
相较于 Original 版本(来自Black Hat Go)
该版本解决了第一次运行时出现的无法运行的意外情况(推测可能由代理, Defender等因素引起, 长时间没有任何反应, 但是几天后运行, 能够正常运行)
该版本使用了一种并发方式, 通过通信来共享内存

在编写Go程序的时候, 需要注意死锁, 缓冲, 同步问题

*/

package main

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

func worker(ports, results chan int, wg *sync.WaitGroup) {
	for p := range ports {
		// fmt.Printf("Scanning port: %d\n", p) // 增加心跳输出
		address := fmt.Sprintf("scanme.namp.org:%d", p)
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err != nil {
			results <- 0 // 失败发送0
			continue
		}
		conn.Close()
		results <- p // 成功发送端口号
	}
	wg.Done()
}

func main() {
	ports := make(chan int, 100) // 给一点缓冲
	results := make(chan int, 100)
	var wg sync.WaitGroup
	var openPorts []int

	// 1. 启动 Workers
	for i := 0; i < 100; i++ { // 假设开启100个并发
		wg.Add(1)
		go worker(ports, results, &wg)
	}

	// 2. 在另一个 Goroutine 中发送任务，发送完就关掉 ports
	go func() {
		for i := 1; i <= 1024; i++ {
			ports <- i
		}
		close(ports)
	}()

	// 3. 关键：另起一个协程收集结果，防止主进程在 Wait 时被 results 阻塞
	// 这个 slice 只能在这个协程里操作，或者等待结束后再处理
	finished := make(chan bool)
	go func() {
		for res := range results {
			if res != 0 {
				openPorts = append(openPorts, res)
			}
		}
		finished <- true
	}()

	// 4. 等待所有 worker 完成扫描
	wg.Wait()
	close(results) // 只有关闭了 results，上面的 range results 才会结束
	<-finished     // 确认结果已经全部存入 openPorts 切片

	// 5. 使用 sort 包进行排序
	sort.Ints(openPorts)

	// 6. 最终打印
	fmt.Println("\n--- Scan Results (Sorted) ---")
	for _, port := range openPorts {
		fmt.Printf("%d is open\n", port)
	}
}