use std::io::{self, BufRead, Write};
use std::fs::File;
use std::collections::HashMap;

fn main() -> io::Result<()> {
    let stdin = io::stdin();
    let reader = stdin.lock();

    let mut files: HashMap<char, File> = HashMap::new();

    for line in reader.lines() {
        let line = line?;
        if let Some(first_char) = line.chars().next().filter(|c| c.is_alphanumeric()) {
            let file = files.entry(first_char)
                            .or_insert_with(|| File::create(format!("file_{}.txt", first_char)).unwrap());
            writeln!(file, "{}", line)?;
        }
    }

    Ok(())
}
