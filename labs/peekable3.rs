use tokio::io::{self, AsyncBufReadExt, BufReader};
use tokio::stream::StreamExt;
use tokio::time::{self, Duration};

#[tokio::main]
async fn main() {
    let stdin = io::stdin(); // 获取标准输入的异步句柄
    let reader = BufReader::new(stdin);
    let mut lines = reader.lines();

    while let Some(line) = lines.next().await {
        let line = line.expect("Failed to read line");
        println!("Received: {}", line);

        // 设置超时检测，例如5秒无数据视为一波结束
        if time::timeout(Duration::from_secs(5), lines.next()).await.is_err() {
            println!("Timeout, assuming no more data in this batch.");
            break; // 超时则退出循环
        }
    }
}
