package upload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// B2Config represents the configuration for Backblaze B2
type B2Config struct {
	KeyID      string
	AppKey     string
	BucketName string
	Prefix     string
	Concurrent int
}

// UploadResult represents the result of an upload operation
type UploadResult struct {
	LocalPath   string
	RemotePath  string
	URL         string
	Size        int64
	ContentType string
	SHA1        string
	UploadedAt  time.Time
	ElapsedTime time.Duration
	Error       error
}

// B2Uploader handles file uploads to Backblaze B2
type B2Uploader struct {
	config B2Config
	client *b2Client
	wg     sync.WaitGroup
	mutex  sync.Mutex
	queue  chan uploadTask
	done   chan struct{}
}

type uploadTask struct {
	localPath  string
	remotePath string
	resultChan chan *UploadResult
}

// NewB2Uploader creates a new Backblaze B2 uploader
func NewB2Uploader(config B2Config) (*B2Uploader, error) {
	// Validate config
	if config.KeyID == "" {
		return nil, errors.New("B2 Key ID is required")
	}
	if config.AppKey == "" {
		return nil, errors.New("B2 Application Key is required")
	}
	if config.BucketName == "" {
		return nil, errors.New("B2 Bucket Name is required")
	}

	// Set default concurrency
	if config.Concurrent <= 0 {
		config.Concurrent = 4
	}

	// Create a new B2 client
	client, err := newB2Client(config.KeyID, config.AppKey, config.BucketName)
	if err != nil {
		return nil, err
	}

	uploader := &B2Uploader{
		config: config,
		client: client,
		queue:  make(chan uploadTask, 100),
		done:   make(chan struct{}),
	}

	// Start worker goroutines
	for i := 0; i < config.Concurrent; i++ {
		go uploader.worker()
	}

	return uploader, nil
}

// Upload uploads a file to B2
func (u *B2Uploader) Upload(ctx context.Context, localPath string) (*UploadResult, error) {
	// Check if file exists
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if fileInfo.IsDir() {
		return nil, errors.New("directories cannot be uploaded directly")
	}

	// Generate remote path
	remotePath := u.generateRemotePath(localPath)

	// Create a channel for the result
	resultChan := make(chan *UploadResult, 1)

	// Add task to queue
	select {
	case u.queue <- uploadTask{localPath, remotePath, resultChan}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Wait for result or context cancellation
	select {
	case result := <-resultChan:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// UploadBatch uploads multiple files to B2
func (u *B2Uploader) UploadBatch(ctx context.Context, localPaths []string) ([]*UploadResult, error) {
	results := make([]*UploadResult, len(localPaths))
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var firstErr error

	for i, localPath := range localPaths {
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			result, err := u.Upload(ctx, path)

			mutex.Lock()
			defer mutex.Unlock()

			results[idx] = result
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}(i, localPath)
	}

	wg.Wait()
	return results, firstErr
}

// Close closes the uploader
func (u *B2Uploader) Close() error {
	close(u.done)
	return nil
}

// worker processes upload tasks
func (u *B2Uploader) worker() {
	for {
		select {
		case task := <-u.queue:
			result := u.processUpload(task.localPath, task.remotePath)
			task.resultChan <- result
		case <-u.done:
			return
		}
	}
}

// processUpload uploads a file to B2
func (u *B2Uploader) processUpload(localPath, remotePath string) *UploadResult {
	startTime := time.Now()

	result := &UploadResult{
		LocalPath:  localPath,
		RemotePath: remotePath,
		UploadedAt: startTime,
	}

	// Get file info
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to stat file: %w", err)
		return result
	}
	result.Size = fileInfo.Size()

	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to open file: %w", err)
		return result
	}
	defer file.Close()

	// In a real implementation, this would call the B2 API
	// This is a placeholder implementation
	// Simulating a successful upload
	time.Sleep(time.Duration(fileInfo.Size()/1000000) * time.Millisecond) // Simulate upload time based on file size

	url := fmt.Sprintf("https://f000.backblazeb2.com/file/%s/%s", u.config.BucketName, remotePath)

	result.URL = url
	result.ContentType = detectContentType(localPath)
	result.SHA1 = "placeholder-sha1-hash"
	result.ElapsedTime = time.Since(startTime)

	return result
}

// generateRemotePath generates a remote path for the file
func (u *B2Uploader) generateRemotePath(localPath string) string {
	// Extract the base name
	fileName := filepath.Base(localPath)

	// Combine with prefix if provided
	if u.config.Prefix != "" {
		return filepath.Join(u.config.Prefix, fileName)
	}

	return fileName
}

// detectContentType detects the content type of a file
func detectContentType(path string) string {
	// Simple content type detection based on extension
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}

// b2Client is a placeholder for a real B2 client
// In a real implementation, this would use the B2 API
type b2Client struct {
	keyID      string
	appKey     string
	bucketName string
	authToken  string
	apiURL     string
	bucketID   string
}

// newB2Client creates a new B2 client
func newB2Client(keyID, appKey, bucketName string) (*b2Client, error) {
	// In a real implementation, this would authenticate with B2
	client := &b2Client{
		keyID:      keyID,
		appKey:     appKey,
		bucketName: bucketName,
		authToken:  "placeholder-auth-token",
		apiURL:     "https://api.backblazeb2.com",
		bucketID:   "placeholder-bucket-id",
	}

	return client, nil
}

// In a real implementation, this would have methods for interacting with the B2 API:
// - authorizeAccount
// - getBucket
// - getUploadURL
// - uploadFile
// - etc.
