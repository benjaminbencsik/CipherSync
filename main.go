package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	// 1. Define and parse command-line flags
	dir := flag.String("dir", ".", "Directory path to process")
	secret := flag.String("key", "", "Secret key for encryption/decryption")
	decrypt := flag.Bool("decrypt", false, "Set to true to decrypt files")
	workers := flag.Int("workers", 4, "Number of worker goroutines")
	flag.Parse()

	if *secret == "" {
		fmt.Println("Error: The 'key' flag is required.")
		os.Exit(1)
	}

	// 2. Setup Worker Pool
	var wg sync.WaitGroup
	jobs := make(chan string)

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(&wg, jobs, *secret, *decrypt)
	}

	// 3. Producer: Walk directory and send files to jobs channel
	go func() {
		filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("Error accessing path %q: %v\n", path, err)
				return err
			}
			if !info.IsDir() {
				if *decrypt {
					if strings.HasSuffix(path, ".enc") {
						jobs <- path
					}
				} else {
					if strings.HasSuffix(path, ".txt") || strings.HasSuffix(path, ".md") {
						jobs <- path
					}
				}
			}
			return nil
		})
		close(jobs) // Close channel when all files have been sent
	}()

	wg.Wait() // Wait for all workers to finish
	fmt.Println("Processing complete.")
}

// worker function defines a consumer in the worker pool
func worker(wg *sync.WaitGroup, jobs <-chan string, secret string, decrypt bool) {
	defer wg.Done()
	key := createKeyHash(secret)
	for path := range jobs {
		var err error
		if decrypt {
			err = decryptFile(path, key)
		} else {
			err = encryptFile(path, key)
		}

		if err != nil {
			fmt.Printf("Failed to process file %s: %v\n", path, err)
		} else {
			fmt.Printf("Successfully processed file: %s\n", path)
		}
	}
}

// createKeyHash generates a 32-byte key from a secret string using SHA-256
func createKeyHash(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}

// encryptFile encrypts a file using AES-GCM
func encryptFile(path string, key []byte) error {
	plaintext, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Write the encrypted file with a .enc extension
	if err = ioutil.WriteFile(path+".enc", ciphertext, 0644); err != nil {
		return err
	}

	// Remove the original plaintext file
	return os.Remove(path)
}

// decryptFile decrypts a file using AES-GCM
func decryptFile(path string, key []byte) error {
	ciphertext, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	// Write the decrypted file, removing the .enc extension
	originalPath := strings.TrimSuffix(path, ".enc")
	if err = ioutil.WriteFile(originalPath, plaintext, 0644); err != nil {
		return err
	}
	
	// Remove the encrypted .enc file
	return os.Remove(path)
}
