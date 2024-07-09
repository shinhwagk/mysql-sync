import yaml
import hcl2
import os

def convert_yaml_to_hcl(yaml_file, hcl_file):
    # 读取 YAML 文件
    with open(yaml_file, 'r') as file:
        yaml_data = yaml.safe_load(file)

    # 转换逻辑
    # 假设 YAML 数据结构与 HCL 期望结构大致相同，这里可能需要根据实际情况调整
    hcl_data = {
        "job": {
            "name": yaml_data["name"],
            "region": yaml_data.get("region", "global"),
            "datacenters": yaml_data["datacenters"],
            "group": [{
                "name": yaml_data["group"]["name"],
                "count": yaml_data["group"]["count"],
                "task": [{
                    "name": yaml_data["group"]["task"]["name"],
                    "driver": yaml_data["group"]["task"]["driver"],
                    "config": yaml_data["group"]["task"]["config"],
                }]
            }]
        }
    }

    # 写入 HCL 文件
    with open(hcl_file, 'w') as file:
        file.write(hcl2.dumps(hcl_data))

# 遍历当前目录的所有 YAML 文件并转换它们
for filename in os.listdir('.'):
    if filename.endswith('.yaml') or filename.endswith('.yml'):
        hcl_filename = filename.split('.')[0] + '.hcl'
        convert_yaml_to_hcl(filename, hcl_filename)
        print(f'Converted {filename} to {hcl_filename}')
