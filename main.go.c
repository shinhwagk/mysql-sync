#include <stdio.h>
#include <stdlib.h>
#include <string.h>

void processBinlogEntry(char *binlogEntry) {
    if (strncmp(binlogEntry, "#", 1) == 0) {
        if (strncmp(binlogEntry, "# at ", 5) == 0) {
            // Specific processing for "# at "
        } else if (strncmp(binlogEntry, "#2", 2) == 0) {
            char *tokens[20];
            int idx = 0;
            char *token = strtok(binlogEntry, " ");
            while (token != NULL && idx < 20) {
                tokens[idx++] = token;
                token = strtok(NULL, " ");
            }
            if (idx > 9 && strcmp(tokens[9], "Write_rows") == 0) {

            }
        } else if (strncmp(binlogEntry, "#700101  ", 9) == 0) {

        }
    } else {
        if (strncmp(binlogEntry, "SET @@SESSION.GTID_NEXT= '", 26) == 0 &&
            strstr(binlogEntry, "'/*!*/;") == binlogEntry + strlen(binlogEntry) - 7) {

        } else if (strcmp(binlogEntry, "COMMIT/*!*/;") == 0 || strcmp(binlogEntry, "ROLLBACK/*!*/;") == 0) {

        } else if (strncmp(binlogEntry, "SET TIMESTAMP=", 14) == 0 &&
                   strstr(binlogEntry, "'/*!*/;") == binlogEntry + strlen(binlogEntry) - 7) {

        } else if (strncmp(binlogEntry, "BINLOG '", 8) == 0) {

        } else {
            printf("%s\n", binlogEntry);
        }

        fprintf(stdout, "%s\n", binlogEntry);
    }
}

int main() {
    char *line = NULL;
    size_t len = 0;
    ssize_t read;

    while ((read = getline(&line, &len, stdin)) != -1) {
        // Remove newline character if present
        if (read > 0 && line[read - 1] == '\n') {
            line[read - 1] = '\0';
        }
        processBinlogEntry(line);
    }

    free(line);  // Free the dynamically allocated line buffer

    if (ferror(stdin)) {
        fprintf(stderr, "Error reading from standard input\n");
        return EXIT_FAILURE;
    }

    return EXIT_SUCCESS;
}
