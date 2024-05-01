import { readLines } from "https://deno.land/std/io/mod.ts";

// 定义一个异步函数处理每行输入

async function processInput() {
  console.log("Please enter some lines of text (Ctrl-D to end):");

  // 创建一个从标准输入读取的迭代器
  for await (const line of readLines(Deno.stdin)) {
    handleLine(line); // 处理每一行
  }

  console.log("Finished reading from standard input.");
}
// 定义处理每行数据的函数
function handleLine(line: string) {
  console.log(`You wrote: ${line}`);
}
// 调用处理函数
processInput();
