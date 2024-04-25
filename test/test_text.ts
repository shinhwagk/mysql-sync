const decoder = new TextDecoder();
const encoder = new TextEncoder();

Deno.stdout.write(encoder.encode("abc\n"));
Deno.stdout.write(encoder.encode("ccc\n"));
