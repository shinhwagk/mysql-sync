const { run } = Deno;

async function pipeCommands() {
  const p1 = run({
    cmd: ["ls"],
    stdout: "piped",  
  });

  const p2 = run({
    cmd: ["grep", "txt"],
    stdin: "piped",    
    stdout: "piped",
  });

  // 管道连接
  await p1.stdout.pipeTo(p2.stdin);

  // 获取第二个命令的输出
  const output = await p2.output();
  console.log(new TextDecoder().decode(output));

  // 等待两个进程结束并检查结果
  const [status1, status2] = await Promise.all([p1.status(), p2.status()]);

  // 输出执行状态
  console.log(`ls command finished with exit code ${status1.code}`);
  console.log(`grep command finished with exit code ${status2.code}`);

  // 关闭所有打开的资源
  p1.stdout.close();
  p2.stdin.close();
  p2.stdout.close();
}

pipeCommands();
