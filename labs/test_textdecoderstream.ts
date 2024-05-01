const decoder = new TextDecoder("utf-8");
const partialData = new Uint8Array([0xF0, 0x9F, 0x92]); // 假设这是一个 4 字节的 emoji 表情的前三个字节
const remainingData = new Uint8Array([0xA9]); // 这是那个 emoji 表情的最后一个字节

// 使用 decode 方法时传入 { stream: true }
console.log((decoder.decode(partialData, { stream: true })).length); // 输出为空，因为字符还未完整 空字符串
console.log(decoder.decode(remainingData, { stream: true })); // 此时输出完整的 emoji 表情
console.log(decoder.decode(new Uint8Array(), { stream: true })); // 确保所有内部缓冲被处理完毕
