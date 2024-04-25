#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <regex.h>







struct StringFunctionPair {
    char *string;
    void (*function)(char *);
};

void process_at(char *str) {
    printf("处理字符串 \"%s\"，这是 # at 的处理函数。\n", str);
}

void process_bt(char *str) {
    printf("处理字符串 \"%s\"，这是 # bt 的处理函数。\n", str);
}

void process_ct(char *str) {
    printf("处理字符串 \"%s\"，这是 # ct 的处理函数。\n", str);
}

// RegexHandler handlers[6];

// void compileRegexes() {
//     &handlers[0].regex = "# at " 
//     handlers[0].handler = handleRegex2;

//     // regcomp(&handlers[1].regex, "^SET TIMESTAMP=[0-9]{10}(?:\\.[0-9]{6})?/\\*!*\\*/;$", REG_EXTENDED);
//     // handlers[1].handler = handleRegex1;

//     // regcomp(&handlers[2].regex, "^SET @@SESSION\\.GTID_NEXT= '([0-9a-z]{8}-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{12}:[0-9]+)'", REG_EXTENDED);
//     // handlers[2].handler = handleRegex3;

//     // regcomp(&handlers[3].regex, "^COMMIT/\\*!*\\*/;$", REG_EXTENDED);
//     // handlers[3].handler = handleRegex3;

//     // regcomp(&handlers[4].regex, "^ROLLBACK/\\*!*\\*/;$", REG_EXTENDED);
//     // handlers[4].handler = handleRegex3;

//     // regcomp(&handlers[5].regex, " Rotate to ", REG_EXTENDED);
//     // handlers[5].handler = handleRegex3;
// }

// void releaseRegexes() {
//     for (int i = 0; i < 6; i++) {
//         regfree(&handlers[i].regex);
//     }
// }
// void releaseRegexes() {
//     for (int i = 0; i < sizeof(binlogEntryAnalyzer) / sizeof(RegexHandler); i++) {
//         regfree(&binlogEntryAnalyzer[i].regex);
//     }
// }

// void processLine(const char *line) {
//     for (int i = 0; i < sizeof(binlogEntryAnalyzer) / sizeof(RegexHandler); i++) {
//         if (!regexec(&binlogEntryAnalyzer[i].regex, line, 0, NULL, 0)) {
//             binlogEntryAnalyzer[i].handler(line);
//             break;
//         }
//     }
// }
void chomp(char *s) {
    s[strcspn(s, "\n")] = '\0';
}

char* chomp_and_copy(const char *s) {
    if (s == NULL) return NULL;

    char *new_s = strdup(s);
    if (new_s == NULL) {
        fprintf(stderr, "Memory allocation failed\n");
        return NULL;
    }

    chomp(new_s);

    return new_s;
}

int main() {


    struct StringFunctionPair pairs[] = {
        {"# at ", process_at},
        {"COMMIT/*!*/;", process_bt},
        {"ROLLBACK/*!*/;", process_ct},
        {"SET @@SESSION.GTID_NEXT= '", process_ct},
        {"SET TIMESTAMP=",process_ct},
        {"#2",process_ct},
        {"#7",process_ct},
        {"BINLOG '",process_ct}
    };

    char *line = NULL;   
    size_t len = 0;      
    ssize_t read;
    int ret;

    // regex_t regex;

    // char *pattern = "^# at [0-9]+$";
    // ret = regcomp(&regex, pattern, REG_EXTENDED);
    // if (ret) {
    //     fprintf(stderr, "Could not compile regex\n");
    //     exit(1);
    // }


    while ((read = getline(&line, &len, stdin)) != -1) {

        char *processed_str = strdup(line);

        chomp(processed_str);



        for (int i = 0; i < sizeof(pairs) / sizeof(pairs[0]); i++) {

            int result = strncmp(processed_str, pairs[i].string, strlen(pairs[i].string));
            if (result == 0) {

                pairs[i].function(processed_str);
                break; 
            }
        }
            
 

    // for (int i = 0; i <   sizeof(handlers)/sizeof(handlers[0 ]); i++) {
    //         if (!regexec(&handlers[i].regex, processed_str, 0, NULL, 0)) {
    //          handlers[i].handler(line);
    //          break;
    //         }
    //     }
        free(processed_str);


        if (line[0] == '#'){
            continue;
        }

        // fprintf(stdout, "%s\n", line);
    }

    free(line); // 释放 getline 分配的内存


    return 0;
}