# Duplicate File Finder (dff)

A duplicate file finder written on Go.

## Features

    - Scan a file tree and find duplicate files by
        - Name
        - Content (via a checksum)
    - Use filters using glob patters
    - Pipe list of files into other programs
        - Like `rm`, or `mv`, etc
    - Use some concurrency features to make it not so slow