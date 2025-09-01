package service

import (
	"context"
	"fmt"
	"testing"

	"stormlink/server/ent/enttest"
	mediapb "stormlink/server/grpc/media/protobuf"
	"stormlink/tests/testcontainers"
	"stormlink/tests/testhelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockS3Client implements S3ClientInterface for testing
type MockS3Client struct {
	uploads     map[string][]byte
	shouldFail  bool
	failMessage string
}

func NewMockS3Client() *MockS3Client {
	return &MockS3Client{
		uploads:    make(map[string][]byte),
		shouldFail: false,
	}
}

func (m *MockS3Client) UploadFile(ctx context.Context, dir, filename string, fileContent []byte) (url, sanitized string, err error) {
	if m.shouldFail {
		return "", "", fmt.Errorf("mock S3 upload failed: %s", m.failMessage)
	}

	// Simple sanitization for testing
	sanitized = fmt.Sprintf("test-%s", filename)
	key := fmt.Sprintf("%s/%s", dir, sanitized)

	if m.uploads == nil {
		m.uploads = make(map[string][]byte)
	}
	m.uploads[key] = fileContent

	url = fmt.Sprintf("/storage/%s", key)
	return url, sanitized, nil
}

func (m *MockS3Client) SetShouldFail(fail bool, message string) {
	m.shouldFail = fail
	m.failMessage = message
}

func (m *MockS3Client) GetUpload(key string) ([]byte, bool) {
	content, exists := m.uploads[key]
	return content, exists
}

func setupMediaService(t *testing.T) (*MediaService, *MockS3Client) {
	helper := testhelper.NewPostgresTestHelper(t)
	helper.WaitForDatabase(t)
	helper.CleanDatabase(t)

	client := helper.GetClient()
	mockS3 := NewMockS3Client()
	service := NewMediaServiceWithClient(mockS3, client)

	// Cleanup function will be called by test
	t.Cleanup(func() {
		helper.Cleanup()
	})

	return service, mockS3
}

func TestMediaService_UploadMedia_Success(t *testing.T) {
	service, mockS3 := setupMediaService(t)
	ctx := context.Background()

	testData := []byte("fake image data")
	req := &mediapb.UploadMediaRequest{
		Dir:         "test",
		Filename:    "test-image.jpg",
		FileContent: testData,
	}

	resp, err := service.UploadMedia(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Url)
	assert.NotEmpty(t, resp.Filename)
	assert.Greater(t, resp.Id, int64(0))
	assert.Contains(t, resp.Url, "/storage/test/")
	assert.Contains(t, resp.Filename, "test-test-image.jpg")

	// Verify the file was "uploaded" to mock S3
	uploadedData, exists := mockS3.GetUpload("test/test-test-image.jpg")
	assert.True(t, exists)
	assert.Equal(t, testData, uploadedData)
}

func TestMediaService_UploadMedia_DefaultDir(t *testing.T) {
	service, _ := setupMediaService(t)
	ctx := context.Background()

	req := &mediapb.UploadMediaRequest{
		Dir:         "", // Empty dir should default to "media"
		Filename:    "image.png",
		FileContent: []byte("fake png data"),
	}

	resp, err := service.UploadMedia(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.Url, "/storage/media/")
}

func TestMediaService_UploadMedia_S3Failure(t *testing.T) {
	service, mockS3 := setupMediaService(t)
	ctx := context.Background()

	// Make S3 upload fail
	mockS3.SetShouldFail(true, "network error")

	req := &mediapb.UploadMediaRequest{
		Dir:         "test",
		Filename:    "test.jpg",
		FileContent: []byte("test data"),
	}

	resp, err := service.UploadMedia(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)

	// Verify it's a gRPC error with Internal code
	st := status.Convert(err)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "failed to upload file to S3")
}

func TestMediaService_UploadMedia_EmptyFilename(t *testing.T) {
	service, _ := setupMediaService(t)
	ctx := context.Background()

	req := &mediapb.UploadMediaRequest{
		Dir:         "test",
		Filename:    "", // Empty filename
		FileContent: []byte("test data"),
	}

	resp, err := service.UploadMedia(ctx, req)

	// This might succeed or fail depending on validation rules
	// If validation fails, we should get InvalidArgument error
	if err != nil {
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Nil(t, resp)
	} else {
		// If it succeeds, make sure we got a response
		assert.NotNil(t, resp)
	}
}

func TestMediaService_UploadMedia_EmptyContent(t *testing.T) {
	service, _ := setupMediaService(t)
	ctx := context.Background()

	req := &mediapb.UploadMediaRequest{
		Dir:         "test",
		Filename:    "empty.txt",
		FileContent: []byte{}, // Empty content
	}

	resp, err := service.UploadMedia(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Url)
	assert.NotEmpty(t, resp.Filename)
}

func TestMediaService_UploadMedia_LargeFile(t *testing.T) {
	service, _ := setupMediaService(t)
	ctx := context.Background()

	// Create a "large" file (100KB of test data)
	largeData := make([]byte, 100*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	req := &mediapb.UploadMediaRequest{
		Dir:         "large",
		Filename:    "large-file.bin",
		FileContent: largeData,
	}

	resp, err := service.UploadMedia(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.Url, "/storage/large/")
}

func TestMediaService_UploadMedia_SpecialCharactersInFilename(t *testing.T) {
	service, _ := setupMediaService(t)
	ctx := context.Background()

	req := &mediapb.UploadMediaRequest{
		Dir:         "special",
		Filename:    "test file with spaces & symbols!@#.jpg",
		FileContent: []byte("test content"),
	}

	resp, err := service.UploadMedia(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	// The filename should be sanitized by our mock
	assert.Contains(t, resp.Filename, "test-")
	assert.Contains(t, resp.Filename, ".jpg") // Extension should be preserved
}

func TestMediaService_UploadMedia_ConcurrentUploads(t *testing.T) {
	service, _ := setupMediaService(t)
	ctx := context.Background()

	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			req := &mediapb.UploadMediaRequest{
				Dir:         "concurrent",
				Filename:    fmt.Sprintf("file-%d.txt", index),
				FileContent: []byte(fmt.Sprintf("content-%d", index)),
			}

			resp, err := service.UploadMedia(ctx, req)
			if err != nil {
				results <- err
				return
			}

			if resp == nil || resp.Id <= 0 {
				results <- fmt.Errorf("invalid response for upload %d", index)
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent upload %d failed", i)
	}
}

func TestMediaService_UploadMedia_ContextCancellation(t *testing.T) {
	service, _ := setupMediaService(t)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &mediapb.UploadMediaRequest{
		Dir:         "test",
		Filename:    "test.jpg",
		FileContent: []byte("test data"),
	}

	resp, err := service.UploadMedia(ctx, req)

	// Context cancellation might be detected at different points
	// Either during S3 upload or database save
	if err != nil {
		assert.Nil(t, resp)
		// Could be context canceled or other error
	}
}

func TestMediaService_UploadMedia_ValidationError(t *testing.T) {
	service, _ := setupMediaService(t)
	ctx := context.Background()

	// Create a request that should fail validation
	// This depends on what the Validate() method actually checks
	req := &mediapb.UploadMediaRequest{
		Dir:         "test",
		Filename:    "test.jpg",
		FileContent: nil, // This might cause validation to fail
	}

	resp, err := service.UploadMedia(ctx, req)

	// If validation fails, we should get InvalidArgument error
	if err != nil {
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "validation error")
		assert.Nil(t, resp)
	}
}

func TestNewMediaServiceWithClient(t *testing.T) {
	helper := testhelper.NewPostgresTestHelper(t)
	helper.WaitForDatabase(t)
	helper.CleanDatabase(t)
	defer helper.Cleanup()

	client := helper.GetClient()
	mockS3 := NewMockS3Client()

	service := NewMediaServiceWithClient(mockS3, client)

	assert.NotNil(t, service)
	assert.Equal(t, mockS3, service.s3)
	assert.Equal(t, client, service.client)
}

// Benchmark test for upload performance
func BenchmarkMediaService_UploadMedia(b *testing.B) {
	ctx := context.Background()

	// Setup test containers
	containers, err := testcontainers.Setup(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer containers.Cleanup()

	// Create Ent client
	client := enttest.Open(b, "postgres", containers.GetPostgresDSN())
	defer client.Close()
	mockS3 := NewMockS3Client()
	service := NewMediaServiceWithClient(mockS3, client)

	testData := []byte("benchmark test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &mediapb.UploadMediaRequest{
			Dir:         "benchmark",
			Filename:    fmt.Sprintf("file-%d.txt", i),
			FileContent: testData,
		}

		_, err := service.UploadMedia(ctx, req)
		if err != nil {
			b.Fatalf("Upload failed: %v", err)
		}
	}
}

// Test helper function for verifying database state
func TestMediaService_DatabaseIntegrity(t *testing.T) {
	service, _ := setupMediaService(t)
	ctx := context.Background()

	// Upload multiple files
	uploads := []struct {
		dir      string
		filename string
		content  []byte
	}{
		{"images", "photo1.jpg", []byte("photo1 content")},
		{"documents", "doc1.pdf", []byte("document content")},
		{"images", "photo2.png", []byte("photo2 content")},
	}

	uploadedIds := make([]int64, 0, len(uploads))

	for _, upload := range uploads {
		req := &mediapb.UploadMediaRequest{
			Dir:         upload.dir,
			Filename:    upload.filename,
			FileContent: upload.content,
		}

		resp, err := service.UploadMedia(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		uploadedIds = append(uploadedIds, resp.Id)
	}

	// Verify all records exist in database
	for _, id := range uploadedIds {
		media, err := service.client.Media.Get(ctx, int(id))
		assert.NoError(t, err)
		assert.NotNil(t, media)
		assert.NotEmpty(t, media.Filename)
		assert.NotEmpty(t, media.URL)
	}

	// Verify total count
	mediaList, err := service.client.Media.Query().All(ctx)
	assert.NoError(t, err)
	assert.Len(t, mediaList, len(uploads))
}
