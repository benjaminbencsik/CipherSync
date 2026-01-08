
# CipherSync

CipherSync is a command-line tool written in Go for concurrently encrypting and decrypting files within a directory using AES-GCM. It leverages a worker pool pattern with goroutines to process files quickly and efficiently.

## Features

-   Encrypts all `.txt` and `.md` files in a directory.
-   Decrypts all `.enc` files in a directory.
-   Uses strong AES-256-GCM encryption.
-   Leverages a concurrent worker pool to process multiple files at once.
-   Uses only the Go standard library.

## Usage

First, build the executable:

```bash
go build
```

### To Encrypt

Provide a directory path and a secret key. The tool will find all `.txt` and `.md` files, encrypt them, and save them with a `.enc` extension, removing the originals.

```bash
./ciphersync -dir="/path/to/your/documents" -key="your-secret-password"
```

### To Decrypt

Add the `-decrypt` flag. The tool will find all `.enc` files, decrypt them using the same secret key, and restore the original files.

```bash
./ciphersync -dir="/path/to/your/documents" -key="your-secret-password" -decrypt=true
```

## How the Worker Pool Manages Files

The worker pool pattern is an efficient way to manage concurrent tasks. In CipherSync, the tasks are file encryption or decryption operations. Here’s how it works:

1.  **The Job Queue (`jobs` channel):** A buffered channel is created to act as a queue for all the file paths that need to be processed. This channel allows the main thread to discover files and the worker goroutines to pick them up for processing without interfering with each other.

2.  **The Producer (File Discovery):** The `main` goroutine acts as the "producer." It walks the specified directory using `filepath.Walk`. When it finds a file that matches the criteria (e.g., a `.txt` file for encryption), it sends the file path as a "job" into the `jobs` channel. After scanning the entire directory, it closes the `jobs` channel to signal that no more jobs will be added.

3.  **The Consumers (Worker Goroutines):** Before the producer starts, the `main` goroutine launches a fixed number of "worker" goroutines (the default is 4, configurable with the `-workers` flag). These workers are the "consumers." Each worker runs in a loop, listening for new messages on the `jobs` channel.

4.  **Processing and Synchronization:**
    -   As soon as a file path is sent to the `jobs` channel, any available worker will pick it up and begin the encryption or decryption process.
    -   Because multiple workers are running at the same time, several files can be processed concurrently, significantly speeding up the total time required.
    -   A `sync.WaitGroup` is used to ensure the `main` goroutine waits for all workers to finish their jobs before exiting. Each worker signals that it is "done" when the `jobs` channel is empty and closed.

This model effectively decouples the task of finding files from the task of processing them, allowing for scalable and efficient file processing.

