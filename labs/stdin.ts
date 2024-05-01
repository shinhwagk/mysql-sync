// async function processInput() {
//   const decoder = new TextDecoder();
//   const reader = Deno.stdin.readable.getReader();

//   console.log("Please enter some lines of text (Ctrl-D to end):");

//   try {
//     while (true) {
//       const { value, done } = await reader.read();
//       if (done) {
//         break;
//       }
//       const text = decoder.decode(value);
//       const lines = text.split(/\n/);
//       lines.forEach((line) => {
//         if (line) {
//           console.log(`Processed line: ${line}`);
//         }
//       });
//     }
//   } finally {
//     reader.releaseLock();
//   }

//   console.log("Finished reading from standard input.");
// }

// processInput();
// const encoder = new TextEncoder();

// const decoder = new TextDecoder();
// for await (const chunk of Deno.stdin.readable) {
//   const text = decoder.decode(chunk, { stream: true });
//   await Deno.stdout.write(encoder.encode(text));
//   // do something with the text
// }

const decoder = new TextDecoder("utf-8");

async function processStandardInputByLine() {
  let buffer = "";

  for await (const chunk of Deno.stdin.readable) {
    const text = decoder.decode(chunk, { stream: true });

    const testlength = text.length;

    if (testlength === 0) {
      continue;
    }

    const newlineIndex = text.indexOf("\n");

    if (newlineIndex === -1) {
      buffer += text;
    } else {
      buffer += text.substring(0, newlineIndex + 1);
      processLine(buffer);
      buffer = newlineIndex + 2 >= text.length
        ? ""
        : text.substring(newlineIndex + 2);
    }
  }

  if (buffer.length > 0) {
    processLine(buffer);
  }
}
const encoder = new TextEncoder();

function processLine(line: string) {
  Deno.stdout.write(encoder.encode(line));
}

processStandardInputByLine();
