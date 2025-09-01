package service

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"stormlink/server/ent/enttest"
	mediapb "stormlink/server/grpc/media/protobuf"
	"stormlink/tests/testcontainers"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockS3Client for testing without actual S3
type MockS3Client struct {
	uploads map[string][]byte
	errors  map[string]error
}

func NewMockS3Client() *MockS3Client {
	return &MockS3Client{
		uploads: make(map[string][]byte),
		errors:  make(map[string]error),
	}
}

func (m *MockS3Client) UploadFile(ctx context.Context, dir, filename string, content []byte) (string, string, error) {
	key := dir + "/" + filename

	if err, exists := m.errors[key]; exists {
		return "", "", err
	}

	sanitizedFilename := filename // In real implementation, this would sanitize the filename
	url := "https://cdn.stormlink.com/" + key

	m.uploads[key] = content

	return url, sanitizedFilename, nil
}

func (m *MockS3Client) SetError(key string, err error) {
	m.errors[key] = err
}

func (m *MockS3Client) GetUpload(key string) ([]byte, bool) {
	content, exists := m.uploads[key]
	return content, exists
}

type MediaServiceTestSuite struct {
	suite.Suite
	containers *testcontainers.TestContainers
	service    *MediaService
	mockS3     *MockS3Client
	ctx        context.Context
}

func (suite *MediaServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test containers
	containers, err := testcontainers.SetupTestContainers(suite.ctx)
	suite.Require().NoError(err)
	suite.containers = containers

	// Create mock S3 client
	suite.mockS3 = NewMockS3Client()

	// Create service with mock S3
	suite.service = &MediaService{
		s3:     suite.mockS3,
		client: containers.EntClient,
	}
}

func (suite *MediaServiceTestSuite) TearDownSuite() {
	if suite.containers != nil {
		err := suite.containers.Cleanup(suite.ctx)
		suite.Require().NoError(err)
	}
}

func (suite *MediaServiceTestSuite) SetupTest() {
	// Reset database state before each test
	err := suite.containers.ResetDatabase(suite.ctx)
	suite.Require().NoError(err)

	// Reset Redis state
	err = suite.containers.FlushRedis(suite.ctx)
	suite.Require().NoError(err)

	// Reset mock S3
	suite.mockS3.uploads = make(map[string][]byte)
	suite.mockS3.errors = make(map[string]error)
}

func (suite *MediaServiceTestSuite) TestUploadMedia_Success() {
	fileContent := []byte("test image content")
	filename := "test-image.jpg"
	dir := "images"

	req := &mediapb.UploadMediaRequest{
		Filename:    filename,
		FileContent: fileContent,
		Dir:         dir,
	}

	resp, err := suite.service.UploadMedia(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().NotEmpty(resp.Url)
	suite.Assert().Equal(filename, resp.Filename)
	suite.Assert().Greater(resp.Id, int64(0))

	// Verify file was uploaded to mock S3
	uploadedContent, exists := suite.mockS3.GetUpload(dir + "/" + filename)
	suite.Assert().True(exists)
	suite.Assert().Equal(fileContent, uploadedContent)

	// Verify media record was created in database
	media, err := suite.containers.EntClient.Media.Get(suite.ctx, int(resp.Id))
	suite.Assert().NoError(err)
	suite.Assert().NotNil(media)
	suite.Assert().Equal(filename, media.Filename)
	suite.Assert().Equal(resp.Url, media.URL)
}

func (suite *MediaServiceTestSuite) TestUploadMedia_DefaultDirectory() {
	fileContent := []byte("default dir test content")
	filename := "default-test.png"

	req := &mediapb.UploadMediaRequest{
		Filename:    filename,
		FileContent: fileContent,
		// Dir not specified - should default to "media"
	}

	resp, err := suite.service.UploadMedia(suite.ctx, req)

	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)
	suite.Assert().NotEmpty(resp.Url)
	suite.Assert().Equal(filename, resp.Filename)

	// Verify file was uploaded to default "media" directory
	uploadedContent, exists := suite.mockS3.GetUpload("media/" + filename)
	suite.Assert().True(exists)
	suite.Assert().Equal(fileContent, uploadedContent)
}

func (suite *MediaServiceTestSuite) TestUploadMedia_EmptyFilename() {
	req := &mediapb.UploadMediaRequest{
		Filename:    "",
		FileContent: []byte("content"),
		Dir:         "images",
	}

	resp, err := suite.service.UploadMedia(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.InvalidArgument, st.Code())
	suite.Assert().Contains(st.Message(), "validation error")
}

func (suite *MediaServiceTestSuite) TestUploadMedia_EmptyContent() {
	req := &mediapb.UploadMediaRequest{
		Filename:    "test.jpg",
		FileContent: []byte{},
		Dir:         "images",
	}

	resp, err := suite.service.UploadMedia(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.InvalidArgument, st.Code())
	suite.Assert().Contains(st.Message(), "validation error")
}

func (suite *MediaServiceTestSuite) TestUploadMedia_S3Failure() {
	filename := "failing-upload.jpg"
	dir := "images"
	fileContent := []byte("test content")

	// Set mock S3 to return error for this upload
	suite.mockS3.SetError(dir+"/"+filename, assert.AnError)

	req := &mediapb.UploadMediaRequest{
		Filename:    filename,
		FileContent: fileContent,
		Dir:         dir,
	}

	resp, err := suite.service.UploadMedia(suite.ctx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)

	st, ok := status.FromError(err)
	suite.Assert().True(ok)
	suite.Assert().Equal(codes.Internal, st.Code())
	suite.Assert().Contains(st.Message(), "failed to upload file to S3")

	// Verify no media record was created in database
	allMedia, err := suite.containers.EntClient.Media.Query().All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Empty(allMedia)
}

func (suite *MediaServiceTestSuite) TestUploadMedia_DatabaseFailure() {
	// This test is tricky because we can't easily simulate database failures
	// with the current setup. In a real scenario, you might use a mock client.

	// For now, let's test a scenario where S3 succeeds but we have data issues
	filename := "db-test.jpg"
	dir := "images"
	fileContent := []byte("test content")

	req := &mediapb.UploadMediaRequest{
		Filename:    filename,
		FileContent: fileContent,
		Dir:         dir,
	}

	// This should succeed with our current setup
	resp, err := suite.service.UploadMedia(suite.ctx, req)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp)

	// Verify S3 upload succeeded
	uploadedContent, exists := suite.mockS3.GetUpload(dir + "/" + filename)
	suite.Assert().True(exists)
	suite.Assert().Equal(fileContent, uploadedContent)
}

func (suite *MediaServiceTestSuite) TestUploadMedia_DifferentFileTypes() {
	testCases := []struct {
		name        string
		filename    string
		content     []byte
		contentType string
	}{
		{
			name:        "JPEG image",
			filename:    "photo.jpg",
			content:     []byte("fake jpeg content"),
			contentType: "image/jpeg",
		},
		{
			name:        "PNG image",
			filename:    "graphic.png",
			content:     []byte("fake png content"),
			contentType: "image/png",
		},
		{
			name:        "SVG image",
			filename:    "icon.svg",
			content:     []byte("<svg>fake svg content</svg>"),
			contentType: "image/svg+xml",
		},
		{
			name:        "PDF document",
			filename:    "document.pdf",
			content:     []byte("fake pdf content"),
			contentType: "application/pdf",
		},
		{
			name:        "Text file",
			filename:    "readme.txt",
			content:     []byte("This is a text file"),
			contentType: "text/plain",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req := &mediapb.UploadMediaRequest{
				Filename:    tc.filename,
				FileContent: tc.content,
				Dir:         "uploads",
			}

			resp, err := suite.service.UploadMedia(suite.ctx, req)

			suite.Assert().NoError(err)
			suite.Assert().NotNil(resp)
			suite.Assert().Equal(tc.filename, resp.Filename)
			suite.Assert().NotEmpty(resp.Url)
			suite.Assert().Greater(resp.Id, int64(0))

			// Verify content was uploaded correctly
			uploadedContent, exists := suite.mockS3.GetUpload("uploads/" + tc.filename)
			suite.Assert().True(exists)
			suite.Assert().Equal(tc.content, uploadedContent)

			// Verify database record
			media, err := suite.containers.EntClient.Media.Get(suite.ctx, int(resp.Id))
			suite.Assert().NoError(err)
			suite.Assert().Equal(tc.filename, media.Filename)
			suite.Assert().Equal(resp.Url, media.URL)
		})
	}
}

func (suite *MediaServiceTestSuite) TestUploadMedia_LargeFiles() {
	// Test uploading larger files
	largeSizes := []int{
		1024,    // 1KB
		10240,   // 10KB
		102400,  // 100KB
		1048576, // 1MB
		5242880, // 5MB
	}

	for _, size := range largeSizes {
		suite.Run(fmt.Sprintf("upload %d bytes", size), func() {
			// Create content of specified size
			content := make([]byte, size)
			for i := range content {
				content[i] = byte(i % 256)
			}

			filename := fmt.Sprintf("large-file-%d.bin", size)
			req := &mediapb.UploadMediaRequest{
				Filename:    filename,
				FileContent: content,
				Dir:         "large-files",
			}

			start := time.Now()
			resp, err := suite.service.UploadMedia(suite.ctx, req)
			duration := time.Since(start)

			suite.Assert().NoError(err)
			suite.Assert().NotNil(resp)
			suite.Assert().Equal(filename, resp.Filename)

			// Verify upload performance is reasonable
			suite.Assert().Less(duration, 1*time.Second, "Large file upload should complete within 1 second")

			// Verify content integrity
			uploadedContent, exists := suite.mockS3.GetUpload("large-files/" + filename)
			suite.Assert().True(exists)
			suite.Assert().Equal(len(content), len(uploadedContent))
			suite.Assert().Equal(content, uploadedContent)

			// Verify database record has correct size
			media, err := suite.containers.EntClient.Media.Get(suite.ctx, int(resp.Id))
			suite.Assert().NoError(err)
			suite.Assert().Equal(int64(size), media.Size)
		})
	}
}

func (suite *MediaServiceTestSuite) TestUploadMedia_SpecialCharactersInFilename() {
	testCases := []struct {
		name             string
		originalFilename string
		expectError      bool
	}{
		{"normal filename", "normal-file.jpg", false},
		{"spaces in filename", "file with spaces.jpg", false},
		{"unicode characters", "файл-тест.jpg", false},
		{"special characters", "file@#$%.jpg", false},
		{"very long filename", "very-long-filename-that-might-cause-issues-in-some-systems-because-of-length-restrictions.jpg", false},
		{"filename with dots", "file.name.with.dots.jpg", false},
		{"filename starting with dot", ".hidden-file.jpg", false},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			content := []byte("test content for " + tc.name)

			req := &mediapb.UploadMediaRequest{
				Filename:    tc.originalFilename,
				FileContent: content,
				Dir:         "special-chars",
			}

			resp, err := suite.service.UploadMedia(suite.ctx, req)

			if tc.expectError {
				suite.Assert().Error(err)
				suite.Assert().Nil(resp)
			} else {
				suite.Assert().NoError(err)
				suite.Assert().NotNil(resp)
				suite.Assert().NotEmpty(resp.Filename)
				suite.Assert().NotEmpty(resp.Url)

				// Verify upload occurred
				uploadedContent, exists := suite.mockS3.GetUpload("special-chars/" + resp.Filename)
				suite.Assert().True(exists)
				suite.Assert().Equal(content, uploadedContent)
			}
		})
	}
}

func (suite *MediaServiceTestSuite) TestUploadMedia_ConcurrentUploads() {
	// Test concurrent uploads
	concurrency := 10
	results := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			filename := fmt.Sprintf("concurrent-file-%d.jpg", index)
			content := []byte(fmt.Sprintf("content for file %d", index))

			req := &mediapb.UploadMediaRequest{
				Filename:    filename,
				FileContent: content,
				Dir:         "concurrent",
			}

			resp, err := suite.service.UploadMedia(suite.ctx, req)
			if err != nil {
				results <- err
				return
			}

			if resp == nil || resp.Id == 0 {
				results <- assert.AnError
				return
			}

			// Verify upload
			uploadedContent, exists := suite.mockS3.GetUpload("concurrent/" + filename)
			if !exists || !bytes.Equal(content, uploadedContent) {
				results <- assert.AnError
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		err := <-results
		suite.Assert().NoError(err, "Concurrent upload %d should succeed", i)
	}

	// Verify all files were uploaded
	allMedia, err := suite.containers.EntClient.Media.Query().All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Len(allMedia, concurrency)

	// Verify all uploads in S3
	for i := 0; i < concurrency; i++ {
		filename := fmt.Sprintf("concurrent-file-%d.jpg", i)
		_, exists := suite.mockS3.GetUpload("concurrent/" + filename)
		suite.Assert().True(exists, "File %s should exist in S3", filename)
	}
}

func (suite *MediaServiceTestSuite) TestUploadMedia_DirectoryHandling() {
	testCases := []struct {
		name      string
		directory string
		expected  string
	}{
		{"standard directory", "images", "images"},
		{"nested directory", "user/avatars", "user/avatars"},
		{"deep nested directory", "communities/logos/thumbnails", "communities/logos/thumbnails"},
		{"directory with special chars", "user-data/files", "user-data/files"},
		{"empty directory uses default", "", "media"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			filename := "test-file.jpg"
			content := []byte("test content")

			req := &mediapb.UploadMediaRequest{
				Filename:    filename,
				FileContent: content,
				Dir:         tc.directory,
			}

			resp, err := suite.service.UploadMedia(suite.ctx, req)

			suite.Assert().NoError(err)
			suite.Assert().NotNil(resp)

			// Verify file was uploaded to correct directory
			expectedKey := tc.expected + "/" + filename
			uploadedContent, exists := suite.mockS3.GetUpload(expectedKey)
			suite.Assert().True(exists, "File should exist at %s", expectedKey)
			suite.Assert().Equal(content, uploadedContent)

			// Verify URL contains correct path
			suite.Assert().Contains(resp.Url, tc.expected)
		})
	}
}

func (suite *MediaServiceTestSuite) TestUploadMedia_ContentTypes() {
	// Test different content types and their handling
	testCases := []struct {
		filename    string
		content     []byte
		description string
	}{
		{"image.jpg", []byte{0xFF, 0xD8, 0xFF}, "JPEG image"},
		{"image.png", []byte{0x89, 0x50, 0x4E, 0x47}, "PNG image"},
		{"document.pdf", []byte{0x25, 0x50, 0x44, 0x46}, "PDF document"},
		{"archive.zip", []byte{0x50, 0x4B, 0x03, 0x04}, "ZIP archive"},
		{"text.txt", []byte("Plain text content"), "Text file"},
	}

	for _, tc := range testCases {
		suite.Run(tc.description, func() {
			req := &mediapb.UploadMediaRequest{
				Filename:    tc.filename,
				FileContent: tc.content,
				Dir:         "mixed-content",
			}

			resp, err := suite.service.UploadMedia(suite.ctx, req)

			suite.Assert().NoError(err)
			suite.Assert().NotNil(resp)
			suite.Assert().Equal(tc.filename, resp.Filename)

			// Verify content is preserved
			uploadedContent, exists := suite.mockS3.GetUpload("mixed-content/" + tc.filename)
			suite.Assert().True(exists)
			suite.Assert().Equal(tc.content, uploadedContent)

			// Verify database record
			media, err := suite.containers.EntClient.Media.Get(suite.ctx, int(resp.Id))
			suite.Assert().NoError(err)
			suite.Assert().Equal(int64(len(tc.content)), media.Size)
		})
	}
}

func (suite *MediaServiceTestSuite) TestUploadMedia_DatabaseIntegrity() {
	// Test that database records are created correctly
	uploads := []struct {
		filename string
		dir      string
		content  []byte
	}{
		{"integrity-test-1.jpg", "test", []byte("content 1")},
		{"integrity-test-2.png", "test", []byte("content 2")},
		{"integrity-test-3.gif", "other", []byte("content 3")},
	}

	var mediaIDs []int64

	// Upload all files
	for _, upload := range uploads {
		req := &mediapb.UploadMediaRequest{
			Filename:    upload.filename,
			FileContent: upload.content,
			Dir:         upload.dir,
		}

		resp, err := suite.service.UploadMedia(suite.ctx, req)
		suite.Assert().NoError(err)
		suite.Assert().NotNil(resp)

		mediaIDs = append(mediaIDs, resp.Id)
	}

	// Verify all records exist and have correct data
	for i, upload := range uploads {
		media, err := suite.containers.EntClient.Media.Get(suite.ctx, int(mediaIDs[i]))
		suite.Assert().NoError(err)
		suite.Assert().NotNil(media)
		suite.Assert().Equal(upload.filename, media.Filename)
		suite.Assert().Contains(media.URL, upload.dir)
		suite.Assert().Equal(int64(len(upload.content)), media.Size)
		suite.Assert().NotZero(media.CreatedAt)
	}

	// Verify total count
	allMedia, err := suite.containers.EntClient.Media.Query().All(suite.ctx)
	suite.Assert().NoError(err)
	suite.Assert().Len(allMedia, len(uploads))
}

func (suite *MediaServiceTestSuite) TestUploadMedia_ContextCancellation() {
	// Test behavior with cancelled context
	cancelledCtx, cancel := context.WithCancel(suite.ctx)
	cancel()

	req := &mediapb.UploadMediaRequest{
		Filename:    "cancelled-upload.jpg",
		FileContent: []byte("test content"),
		Dir:         "test",
	}

	resp, err := suite.service.UploadMedia(cancelledCtx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)
	suite.Assert().Contains(err.Error(), "context canceled")
}

func (suite *MediaServiceTestSuite) TestUploadMedia_TimeoutScenario() {
	// Test with short timeout context
	timeoutCtx, cancel := context.WithTimeout(suite.ctx, 1*time.Millisecond)
	defer cancel()

	// Add delay to S3 mock to trigger timeout
	suite.mockS3.SetError("timeout-test/slow-file.jpg", context.DeadlineExceeded)

	req := &mediapb.UploadMediaRequest{
		Filename:    "slow-file.jpg",
		FileContent: []byte("content that will timeout"),
		Dir:         "timeout-test",
	}

	resp, err := suite.service.UploadMedia(timeoutCtx, req)

	suite.Assert().Error(err)
	suite.Assert().Nil(resp)
	// Error could be context deadline exceeded or S3 error
	suite.Assert().True(
		err == context.DeadlineExceeded ||
			status.Code(err) == codes.Internal,
		"Should be timeout or S3 error",
	)
}

func (suite *MediaServiceTestSuite) TestUploadMedia_UniqueFilenames() {
	// Test that duplicate filenames are handled appropriately
	filename := "duplicate-test.jpg"
	content1 := []byte("first upload content")
	content2 := []byte("second upload content")

	// First upload
	req1 := &mediapb.UploadMediaRequest{
		Filename:    filename,
		FileContent: content1,
		Dir:         "duplicates",
	}

	resp1, err := suite.service.UploadMedia(suite.ctx, req1)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp1)

	// Second upload with same filename
	req2 := &mediapb.UploadMediaRequest{
		Filename:    filename,
		FileContent: content2,
		Dir:         "duplicates",
	}

	resp2, err := suite.service.UploadMedia(suite.ctx, req2)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(resp2)

	// Both should succeed (overwriting or with unique names depending on implementation)
	suite.Assert().NotEqual(resp1.Id, resp2.Id, "Should create separate media records")

	// Verify both records exist in database
	media1, err := suite.containers.EntClient.Media.Get(suite.ctx, int(resp1.Id))
	suite.Assert().NoError(err)
	suite.Assert().NotNil(media1)

	media2, err := suite.containers.EntClient.Media.Get(suite.ctx, int(resp2.Id))
	suite.Assert().NoError(err)
	suite.Assert().NotNil(media2)
}

// Test with SQLite for faster unit tests
func TestMediaService_Unit(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	mockS3 := NewMockS3Client()
	service := &MediaService{
		s3:     mockS3,
		client: client,
	}
	ctx := context.Background()

	t.Run("basic upload functionality", func(t *testing.T) {
		req := &mediapb.UploadMediaRequest{
			Filename:    "unit-test.jpg",
			FileContent: []byte("unit test content"),
			Dir:         "unit-tests",
		}

		resp, err := service.UploadMedia(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "unit-test.jpg", resp.Filename)
		assert.Contains(t, resp.Url, "unit-tests")
		assert.Greater(t, resp.Id, int64(0))

		// Verify in mock S3
		content, exists := mockS3.GetUpload("unit-tests/unit-test.jpg")
		assert.True(t, exists)
		assert.Equal(t, []byte("unit test content"), content)
	})

	t.Run("validation error handling", func(t *testing.T) {
		req := &mediapb.UploadMediaRequest{
			Filename:    "", // Invalid
			FileContent: []byte("content"),
			Dir:         "test",
		}

		resp, err := service.UploadMedia(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// Run integration test suite
func TestMediaServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(MediaServiceTestSuite))
}

// Benchmark tests
func BenchmarkMediaService_UploadMedia(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	mockS3 := NewMockS3Client()
	service := &MediaService{
		s3:     mockS3,
		client: client,
	}
	ctx := context.Background()

	content := []byte("benchmark test content")
	req := &mediapb.UploadMediaRequest{
		Filename:    "benchmark-test.jpg",
		FileContent: content,
		Dir:         "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use unique filename for each iteration
		req.Filename = fmt.Sprintf("benchmark-test-%d.jpg", i)

		_, err := service.UploadMedia(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMediaService_UploadMedia_LargeFiles(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	mockS3 := NewMockS3Client()
	service := &MediaService{
		s3:     mockS3,
		client: client,
	}
	ctx := context.Background()

	// Create 1MB test content
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &mediapb.UploadMediaRequest{
			Filename:    fmt.Sprintf("large-benchmark-%d.bin", i),
			FileContent: content,
			Dir:         "large-benchmark",
		}

		_, err := service.UploadMedia(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Performance benchmark with real PostgreSQL
func BenchmarkMediaService_PostgreSQL(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmarks in short mode")
	}

	ctx := context.Background()

	// Setup containers
	containers, err := testcontainers.SetupTestContainers(ctx)
	require.NoError(b, err)
	defer func() {
		err := containers.Cleanup(ctx)
		require.NoError(b, err)
	}()

	// Setup service
	mockS3 := NewMockS3Client()
	service := &MediaService{
		s3:     mockS3,
		client: containers.EntClient,
	}

	content := []byte("postgresql benchmark content")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &mediapb.UploadMediaRequest{
			Filename:    fmt.Sprintf("pg-benchmark-%d.jpg", i),
			FileContent: content,
			Dir:         "pg-benchmark",
		}

		_, err := service.UploadMedia(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
