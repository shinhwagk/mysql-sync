const binlogEntryAnalyzer = [
  {
    regex: /^# at [0-9]+$/,
    handler: (line: string) => console.log(`Handling regex2 for line: ${line}`),
  },
  {
    regex: /^SET TIMESTAMP=[0-9]{10}(?:\.[0-9]{6})?\/\*!*\*\/;$/,
    handler: (line: string) => console.log(`Handling regex1 for line: ${line}`),
  },
  {
    regex:
      /^SET @@SESSION\.GTID_NEXT= '([0-9a-z]{8}-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{12}:[0-9]+)'/,
    handler: (line: string) => console.log(`Handling regex3 for line: ${line}`),
  },
  {
    regex: /^COMMIT\/\*!*\*\/;$/,
    handler: (line: string) => console.log(`Handling regex3 for line: ${line}`),
  },
  {
    regex: /^ROLLBACK\/\*!*\*\/;$/,
    handler: (line: string) => console.log(`Handling regex3 for line: ${line}`),
  },
  {
    regex: / Rotate to /,
    handler: (line: string) => console.log(`Handling regex3 for line: ${line}`),
  },
];

async function binlogEntryListener(lines: string[]) {
  for (const line of lines) {
    for (const { regex, handler } of binlogEntryAnalyzer) {
      if (regex.test(line)) {
        handler(line);
        break;
      }
    }
  }
}

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

function processLine(binlogEntry: string) {
  for (const { regex, handler } of binlogEntryAnalyzer) {
    if (regex.test(binlogEntry)) {
      handler(binlogEntry);
      break;
    }
  }
}

processStandardInputByLine();
