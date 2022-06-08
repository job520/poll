# golang 协程池
### 用法
1. 获取包：  
`go get github.com/job520/pool`

2. 示例代码：
    ```go
    package main
    import (
        "fmt"
        "github.com/job520/pool"
    )
    func main() {
        p, err := pool.NewPool(10)
        if err != nil {
            panic(err)
        }
        for i := 0; i < 5; i++ {
            // 将任务放入协程池
            err := p.Put(&pool.Task{
                Handler: func(v ...interface{}) {
                    fmt.Println(v)
                },
                Params: []interface{}{i, "hello"},
            })
            if err != nil {
                fmt.Println("放入协程池失败：" + err.Error())
            }
        }
        // 安全关闭协程池（保证已加入池中的任务被消费完）
        p.Close()
        // 如果协程池已经关闭, Put() 方法会返回错误
        err = p.Put(&pool.Task{
            Handler: func(v ...interface{}) {},
        })
        if err != nil {
            fmt.Println(err)
        }
    }
    ```