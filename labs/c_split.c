#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// 动态创建的字符串数组需要由调用者释放
char** getElements(const char* input, const char* delimiter, const int* indices, size_t num_indices) {
    char* token;
    char* str_copy;
    size_t i, j;
    int count = 0;
    char** results = malloc(num_indices * sizeof(char*));

    if (!results) {
        fprintf(stderr, "Memory allocation failed\n");
        return NULL;
    }


    str_copy = strdup(input);
    if (!str_copy) {
        fprintf(stderr, "Memory allocation failed\n");
        free(results);
        return NULL;
    }

    // 开始分割字符串
    token = strtok(str_copy, delimiter);
    while (token != NULL) {
        // 检查当前元素是否是所需的元素之一
        for (i = 0; i < num_indices; i++) {
            if (count == indices[i]) {
                results[i] = strdup(token);
                if (!results[i]) {
                    fprintf(stderr, "Memory allocation failed\n");
                    // 清理已分配的内存
                    for (j = 0; j < i; j++) {
                        free(results[j]);
                    }
                    free(results);
                    free(str_copy);
                    return NULL;
                }
            }
        }
        count++;
        token = strtok(NULL, delimiter);
    }

    free(str_copy);
    return results;
}

int main() {
    const char* str = "zero one two three four five six seven eight nine";
    const char* delimiter = " ";
    int indices[] = {2, 5, 8}; // 提取第3、6和9个元素
    size_t num_indices = sizeof(indices) / sizeof(indices[0]);

    char** extracted = getElements(str, delimiter, indices, num_indices);
    if (extracted) {
        for (size_t i = 0; i < num_indices; i++) {
            printf("Element %d: %s\n", indices[i], extracted[i]);
            free(extracted[i]);  // 释放单个字符串
        }
        free(extracted);  // 释放字符串数组
    }

    return 0;
}
