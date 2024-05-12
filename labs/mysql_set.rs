fn binary_to_set(binary_str: &str, set_options: &[&str]) -> String {
    // 将二进制字符串转换为数字
    let num = usize::from_str_radix(binary_str, 2).unwrap();

    // 迭代 set_options，使用位运算来检查每一位是否为1
    set_options
        .iter()
        .enumerate()
        .filter(|&(i, _)| (num >> i) & 1 == 1)
        .map(|(_, name)| *name)
        .collect::<Vec<&str>>()
        .join(",")
}

fn main() {
    let set_options = ["set1", "set2", "set3"];
    let binary_str = "00000011"; // 对应 set1 和 set2 被选择

    let result = binary_to_set(binary_str, &set_options);
    println!("Result: '{}'", result); // 应输出 "set1,set2"
}
