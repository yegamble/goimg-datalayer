# Sprint 6 Test Strategy

> Comprehensive testing strategy for Gallery Context application and HTTP layers.
> **Target**: 85%+ application, 75%+ HTTP, 80%+ overall coverage with async processing testing.

## Executive Summary

Sprint 6 implements the application and HTTP layers for gallery functionality. This strategy defines test patterns for:
- **Image upload with async processing** (multipart, validation, background jobs)
- **Album management CRUD** (create, read, update, delete with ownership)
- **Search with PostgreSQL full-text** (tags, titles, filters, pagination)
- **Social features** (likes, comments with authorization)

**Key Metrics:**
- Application layer coverage: **85%+** (commands and queries)
- HTTP handler coverage: **75%+** (handlers and middleware)
- Integration test coverage: **70%+** (repositories with testcontainers)
- E2E test scenarios: **30+ requests** (Newman/Postman collection)

---

## Test Pyramid for Sprint 6

```
                    ┌─────────────────────┐
                    │   E2E Tests         │  15% (~20-25 scenarios)
                    │   Newman/Postman    │  Full upload flow
                    │                     │  Album CRUD workflows
                    ├─────────────────────┤
                    │  Integration Tests  │  25% (~40-50 tests)
                    │  Testcontainers     │  Repository tests
                    │  Asynq Jobs         │  Background processing
                    ├─────────────────────┤
                    │                     │
                    │   Unit Tests        │  60% (~120-150 tests)
                    │   Command Handlers  │  Pure logic
                    │   Query Handlers    │  Mock dependencies
                    │   HTTP Handlers     │  httptest
                    └─────────────────────┘
```

---

## 1. Application Layer Testing (85%+ Coverage)

### 1.1 Command Handler Tests

Sprint 6 introduces commands for image and album management. All command handlers must follow the existing pattern from identity commands.

#### UploadImageCommand Test Pattern

```go
package commands_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/application/gallery/commands"
    "goimg-datalayer/internal/domain/gallery"
    "goimg-datalayer/internal/domain/identity"
)

func TestUploadImageHandler_Handle(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        cmd     commands.UploadImageCommand
        setup   func(t *testing.T, suite *TestSuite)
        wantErr error
        assert  func(t *testing.T, suite *TestSuite, result interface{}, err error)
    }{
        {
            name: "successful upload with valid image",
            cmd: commands.UploadImageCommand{
                OwnerID:  validUserID,
                Data:     validJPEGData,
                Filename: "test.jpg",
                Title:    "Test Image",
                Tags:     []string{"nature", "landscape"},
            },
            setup: func(t *testing.T, suite *TestSuite) {
                // Validator returns valid result
                suite.Validator.On("Validate", mock.Anything, mock.Anything, "test.jpg").
                    Return(&validator.Result{Valid: true, Format: "jpeg"}, nil)

                // Image repository saves successfully
                suite.ImageRepo.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Image")).
                    Return(nil)

                // Job queue enqueues processing job
                suite.JobQueue.On("Enqueue", mock.Anything, mock.AnythingOfType("*jobs.ProcessImageJob")).
                    Return(nil)

                // Event publisher publishes ImageUploaded event
                suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).
                    Return(nil).Maybe()
            },
            wantErr: nil,
            assert: func(t *testing.T, suite *TestSuite, result interface{}, err error) {
                require.NoError(t, err)
                require.NotNil(t, result)

                imageID := result.(gallery.ImageID)
                assert.False(t, imageID.IsEmpty())

                // Verify image was saved with correct status
                suite.ImageRepo.AssertCalled(t, "Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
                    return img.Status() == gallery.StatusProcessing &&
                        img.Metadata().Title() == "Test Image" &&
                        len(img.Tags()) == 2
                }))

                // Verify processing job was enqueued
                suite.JobQueue.AssertCalled(t, "Enqueue", mock.Anything, mock.Anything)
            },
        },
        {
            name: "rejects malware-infected file",
            cmd: commands.UploadImageCommand{
                OwnerID:  validUserID,
                Data:     malwareData,
                Filename: "malware.jpg",
            },
            setup: func(t *testing.T, suite *TestSuite) {
                // Validator detects malware
                suite.Validator.On("Validate", mock.Anything, malwareData, "malware.jpg").
                    Return(&validator.Result{
                        Valid:   false,
                        Errors:  []string{"malware detected"},
                        ScanResult: &clamav.ScanResult{Infected: true, Virus: "EICAR"},
                    }, nil)
            },
            wantErr: nil, // Will be wrapped error
            assert: func(t *testing.T, suite *TestSuite, result interface{}, err error) {
                require.Error(t, err)
                assert.Contains(t, err.Error(), "validation failed")
                assert.Nil(t, result)

                // Verify image was never saved
                suite.ImageRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
                suite.JobQueue.AssertNotCalled(t, "Enqueue", mock.Anything, mock.Anything)
            },
        },
        {
            name: "rejects oversized file",
            cmd: commands.UploadImageCommand{
                OwnerID:  validUserID,
                Data:     make([]byte, 11*1024*1024), // 11MB
                Filename: "huge.jpg",
            },
            setup: func(t *testing.T, suite *TestSuite) {
                suite.Validator.On("Validate", mock.Anything, mock.Anything, "huge.jpg").
                    Return(&validator.Result{
                        Valid:  false,
                        Errors: []string{"file too large"},
                    }, gallery.ErrFileTooLarge)
            },
            wantErr: nil,
            assert: func(t *testing.T, suite *TestSuite, result interface{}, err error) {
                require.Error(t, err)
                assert.ErrorIs(t, err, gallery.ErrFileTooLarge)
                assert.Nil(t, result)
            },
        },
        {
            name: "handles repository save error",
            cmd: commands.UploadImageCommand{
                OwnerID:  validUserID,
                Data:     validJPEGData,
                Filename: "test.jpg",
            },
            setup: func(t *testing.T, suite *TestSuite) {
                suite.Validator.On("Validate", mock.Anything, mock.Anything, "test.jpg").
                    Return(&validator.Result{Valid: true, Format: "jpeg"}, nil)

                // Database error
                suite.ImageRepo.On("Save", mock.Anything, mock.Anything).
                    Return(fmt.Errorf("database connection failed"))
            },
            wantErr: nil,
            assert: func(t *testing.T, suite *TestSuite, result interface{}, err error) {
                require.Error(t, err)
                assert.Contains(t, err.Error(), "save image")
                assert.Nil(t, result)

                // Job should not be enqueued if save fails
                suite.JobQueue.AssertNotCalled(t, "Enqueue", mock.Anything, mock.Anything)
            },
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            // Arrange
            suite := NewTestSuite(t)
            if tt.setup != nil {
                tt.setup(t, suite)
            }

            handler := commands.NewUploadImageHandler(
                suite.ImageRepo,
                suite.Validator,
                suite.JobQueue,
                suite.EventPublisher,
                &suite.Logger,
            )

            // Act
            result, err := handler.Handle(context.Background(), tt.cmd)

            // Assert
            if tt.assert != nil {
                tt.assert(t, suite, result, err)
            }

            suite.AssertExpectations()
        })
    }
}
```

#### Album Command Tests

```go
func TestCreateAlbumHandler_Handle(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        cmd     commands.CreateAlbumCommand
        setup   func(t *testing.T, suite *TestSuite)
        wantErr error
    }{
        {
            name: "creates album successfully",
            cmd: commands.CreateAlbumCommand{
                OwnerID:     validUserID,
                Title:       "Summer Vacation",
                Description: "Photos from summer trip",
                Visibility:  "private",
            },
            setup: func(t *testing.T, suite *TestSuite) {
                suite.AlbumRepo.On("Save", mock.Anything, mock.AnythingOfType("*gallery.Album")).
                    Return(nil)
                suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).
                    Return(nil).Maybe()
            },
            wantErr: nil,
        },
        {
            name: "rejects empty title",
            cmd: commands.CreateAlbumCommand{
                OwnerID: validUserID,
                Title:   "",
            },
            setup: func(t *testing.T, suite *TestSuite) {
                // No repository calls - validation fails early
            },
            wantErr: gallery.ErrTitleRequired,
        },
        {
            name: "rejects title exceeding 255 chars",
            cmd: commands.CreateAlbumCommand{
                OwnerID: validUserID,
                Title:   strings.Repeat("a", 256),
            },
            setup: func(t *testing.T, suite *TestSuite) {
                // No repository calls
            },
            wantErr: gallery.ErrTitleTooLong,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            suite := NewTestSuite(t)
            if tt.setup != nil {
                tt.setup(t, suite)
            }

            handler := commands.NewCreateAlbumHandler(
                suite.AlbumRepo,
                suite.EventPublisher,
                &suite.Logger,
            )

            result, err := handler.Handle(context.Background(), tt.cmd)

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                assert.Nil(t, result)
                suite.AlbumRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, result)
                suite.AlbumRepo.AssertExpectations(t)
            }
        })
    }
}
```

### 1.2 Query Handler Tests

Query handlers for gallery context should focus on data retrieval with filters, sorting, and pagination.

#### SearchImagesQuery Test Pattern

```go
func TestSearchImagesHandler_Handle(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        query   queries.SearchImagesQuery
        setup   func(t *testing.T, suite *TestSuite)
        want    int
        wantErr error
    }{
        {
            name: "searches by title successfully",
            query: queries.SearchImagesQuery{
                Query:      "sunset",
                Visibility: "public",
                Page:       1,
                PageSize:   20,
            },
            setup: func(t *testing.T, suite *TestSuite) {
                expectedImages := []*gallery.Image{
                    createTestImage(t, "Sunset Beach"),
                    createTestImage(t, "Mountain Sunset"),
                }

                suite.ImageRepo.On("Search", mock.Anything, mock.MatchedBy(func(q gallery.SearchQuery) bool {
                    return q.Query == "sunset" && q.Visibility == gallery.VisibilityPublic
                }), mock.Anything).Return(expectedImages, int64(2), nil)
            },
            want:    2,
            wantErr: nil,
        },
        {
            name: "filters by tags",
            query: queries.SearchImagesQuery{
                Tags:     []string{"nature", "landscape"},
                Page:     1,
                PageSize: 20,
            },
            setup: func(t *testing.T, suite *TestSuite) {
                expectedImages := []*gallery.Image{createTestImage(t, "Nature Landscape")}
                suite.ImageRepo.On("Search", mock.Anything, mock.Anything, mock.Anything).
                    Return(expectedImages, int64(1), nil)
            },
            want:    1,
            wantErr: nil,
        },
        {
            name: "returns empty results for no matches",
            query: queries.SearchImagesQuery{
                Query:    "nonexistent",
                Page:     1,
                PageSize: 20,
            },
            setup: func(t *testing.T, suite *TestSuite) {
                suite.ImageRepo.On("Search", mock.Anything, mock.Anything, mock.Anything).
                    Return([]*gallery.Image{}, int64(0), nil)
            },
            want:    0,
            wantErr: nil,
        },
        {
            name: "handles invalid page number",
            query: queries.SearchImagesQuery{
                Query:    "test",
                Page:     0, // Invalid
                PageSize: 20,
            },
            setup:   func(t *testing.T, suite *TestSuite) {},
            wantErr: shared.ErrInvalidPagination,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            suite := NewTestSuite(t)
            if tt.setup != nil {
                tt.setup(t, suite)
            }

            handler := queries.NewSearchImagesHandler(suite.ImageRepo)

            result, err := handler.Handle(context.Background(), tt.query)

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                assert.Nil(t, result)
            } else {
                require.NoError(t, err)
                assert.Len(t, result.Images, tt.want)
                suite.ImageRepo.AssertExpectations(t)
            }
        })
    }
}
```

### 1.3 Test Suite Setup

Create a reusable test suite for gallery application tests:

```go
package commands_test

import (
    "testing"

    "github.com/rs/zerolog"
    "github.com/stretchr/testify/mock"

    "goimg-datalayer/internal/domain/gallery"
    "goimg-datalayer/internal/infrastructure/storage/validator"
)

type TestSuite struct {
    ImageRepo       *MockImageRepository
    AlbumRepo       *MockAlbumRepository
    Validator       *MockImageValidator
    JobQueue        *MockJobQueue
    EventPublisher  *MockEventPublisher
    Logger          zerolog.Logger
}

func NewTestSuite(t *testing.T) *TestSuite {
    t.Helper()

    return &TestSuite{
        ImageRepo:      new(MockImageRepository),
        AlbumRepo:      new(MockAlbumRepository),
        Validator:      new(MockImageValidator),
        JobQueue:       new(MockJobQueue),
        EventPublisher: new(MockEventPublisher),
        Logger:         zerolog.Nop(),
    }
}

func (s *TestSuite) AssertExpectations() {
    s.ImageRepo.AssertExpectations(t)
    s.AlbumRepo.AssertExpectations(t)
    s.Validator.AssertExpectations(t)
    s.JobQueue.AssertExpectations(t)
    s.EventPublisher.AssertExpectations(t)
}

// Mock repositories (using testify/mock)
type MockImageRepository struct {
    mock.Mock
}

func (m *MockImageRepository) Save(ctx context.Context, image *gallery.Image) error {
    args := m.Called(ctx, image)
    return args.Error(0)
}

func (m *MockImageRepository) FindByID(ctx context.Context, id gallery.ImageID) (*gallery.Image, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*gallery.Image), args.Error(1)
}

func (m *MockImageRepository) Search(ctx context.Context, query gallery.SearchQuery, pagination shared.Pagination) ([]*gallery.Image, int64, error) {
    args := m.Called(ctx, query, pagination)
    if args.Get(0) == nil {
        return nil, args.Get(1).(int64), args.Error(2)
    }
    return args.Get(0).([]*gallery.Image), args.Get(1).(int64), args.Error(2)
}

// ... other repository methods

type MockImageValidator struct {
    mock.Mock
}

func (m *MockImageValidator) Validate(ctx context.Context, data []byte, filename string) (*validator.Result, error) {
    args := m.Called(ctx, data, filename)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*validator.Result), args.Error(1)
}

type MockJobQueue struct {
    mock.Mock
}

func (m *MockJobQueue) Enqueue(ctx context.Context, job interface{}) error {
    args := m.Called(ctx, job)
    return args.Error(0)
}
```

---

## 2. Async Processing Testing (Asynq Jobs)

### 2.1 Job Handler Test Pattern

Background jobs for image processing must be thoroughly tested with proper error handling and retry logic.

```go
package jobs_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/application/gallery/jobs"
    "goimg-datalayer/internal/domain/gallery"
)

func TestProcessImageJobHandler_Handle(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        payload jobs.ProcessImagePayload
        setup   func(t *testing.T, suite *JobTestSuite)
        wantErr error
    }{
        {
            name: "processes image successfully",
            payload: jobs.ProcessImagePayload{
                ImageID: validImageID.String(),
                OwnerID: validOwnerID.String(),
            },
            setup: func(t *testing.T, suite *JobTestSuite) {
                image := createTestImage(t, validImageID, gallery.StatusProcessing)

                // Image repository returns processing image
                suite.ImageRepo.On("FindByID", mock.Anything, validImageID).
                    Return(image, nil)

                // Storage retrieves original data
                suite.Storage.On("Get", mock.Anything, mock.Anything).
                    Return(validJPEGData, nil)

                // Processor generates variants
                suite.Processor.On("Process", mock.Anything, validJPEGData, mock.Anything).
                    Return(&processor.ProcessResult{
                        Thumbnail: processor.VariantData{Data: thumbnailData, Width: 160},
                        Small:     processor.VariantData{Data: smallData, Width: 320},
                        Medium:    processor.VariantData{Data: mediumData, Width: 800},
                        Large:     processor.VariantData{Data: largeData, Width: 1600},
                        Original:  processor.VariantData{Data: validJPEGData, Width: 3000},
                    }, nil)

                // Storage saves all variants
                suite.Storage.On("Put", mock.Anything, mock.Anything, mock.Anything).
                    Return(nil).Times(5)

                // Image repository saves updated image with variants
                suite.ImageRepo.On("Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
                    return img.Status() == gallery.StatusActive && len(img.Variants()) == 4
                })).Return(nil)

                // Event publisher publishes ImageProcessed event
                suite.EventPublisher.On("Publish", mock.Anything, mock.Anything).
                    Return(nil).Maybe()
            },
            wantErr: nil,
        },
        {
            name: "marks image as failed on processing error",
            payload: jobs.ProcessImagePayload{
                ImageID: validImageID.String(),
                OwnerID: validOwnerID.String(),
            },
            setup: func(t *testing.T, suite *JobTestSuite) {
                image := createTestImage(t, validImageID, gallery.StatusProcessing)
                suite.ImageRepo.On("FindByID", mock.Anything, validImageID).
                    Return(image, nil)

                suite.Storage.On("Get", mock.Anything, mock.Anything).
                    Return(validJPEGData, nil)

                // Processing fails
                suite.Processor.On("Process", mock.Anything, validJPEGData, mock.Anything).
                    Return(nil, fmt.Errorf("libvips processing error"))

                // Image repository saves failed status
                suite.ImageRepo.On("Save", mock.Anything, mock.MatchedBy(func(img *gallery.Image) bool {
                    return img.Status() == gallery.StatusFailed
                })).Return(nil)
            },
            wantErr: nil, // Error is handled, not returned
        },
        {
            name: "retries on transient storage errors",
            payload: jobs.ProcessImagePayload{
                ImageID: validImageID.String(),
                OwnerID: validOwnerID.String(),
            },
            setup: func(t *testing.T, suite *JobTestSuite) {
                image := createTestImage(t, validImageID, gallery.StatusProcessing)
                suite.ImageRepo.On("FindByID", mock.Anything, validImageID).
                    Return(image, nil)

                // Storage temporarily unavailable
                suite.Storage.On("Get", mock.Anything, mock.Anything).
                    Return(nil, storage.ErrTemporarilyUnavailable)
            },
            wantErr: storage.ErrTemporarilyUnavailable, // Should retry
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            suite := NewJobTestSuite(t)
            if tt.setup != nil {
                tt.setup(t, suite)
            }

            handler := jobs.NewProcessImageJobHandler(
                suite.ImageRepo,
                suite.Storage,
                suite.Processor,
                suite.EventPublisher,
                &suite.Logger,
            )

            err := handler.Handle(context.Background(), tt.payload)

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
            } else {
                require.NoError(t, err)
                suite.ImageRepo.AssertExpectations(t)
                suite.Storage.AssertExpectations(t)
                suite.Processor.AssertExpectations(t)
            }
        })
    }
}
```

### 2.2 Integration Tests with Asynq

Test the full job lifecycle with a real Asynq server and Redis:

```go
package jobs_test

import (
    "context"
    "testing"
    "time"

    "github.com/hibiken/asynq"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go/modules/redis"

    "goimg-datalayer/internal/application/gallery/jobs"
)

func TestProcessImageJob_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    // Start Redis container for Asynq
    redisContainer, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    require.NoError(t, err)
    defer redisContainer.Terminate(ctx)

    redisAddr, err := redisContainer.ConnectionString(ctx)
    require.NoError(t, err)

    // Create Asynq client
    client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
    defer client.Close()

    // Create Asynq server
    srv := asynq.NewServer(
        asynq.RedisClientOpt{Addr: redisAddr},
        asynq.Config{
            Concurrency: 1,
            Queues: map[string]int{
                "default": 10,
            },
        },
    )

    // Register job handler
    mux := asynq.NewServeMux()
    handler := jobs.NewProcessImageJobHandler(
        mockImageRepo,
        mockStorage,
        mockProcessor,
        mockEventPublisher,
        &logger,
    )
    mux.HandleFunc(jobs.TypeProcessImage, handler.Handle)

    // Start server in goroutine
    go func() {
        if err := srv.Run(mux); err != nil {
            t.Logf("asynq server error: %v", err)
        }
    }()
    defer srv.Shutdown()

    // Enqueue job
    payload := jobs.ProcessImagePayload{
        ImageID: validImageID.String(),
        OwnerID: validOwnerID.String(),
    }
    task, err := jobs.NewProcessImageTask(payload)
    require.NoError(t, err)

    info, err := client.Enqueue(task)
    require.NoError(t, err)

    // Wait for job to be processed
    time.Sleep(2 * time.Second)

    // Verify job was processed
    inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: redisAddr})
    defer inspector.Close()

    taskInfo, err := inspector.GetTaskInfo("default", info.ID)
    require.NoError(t, err)
    assert.Equal(t, asynq.TaskStateCompleted, taskInfo.State)
}
```

---

## 3. HTTP Layer Testing (75%+ Coverage)

### 3.1 Upload Handler Test Pattern

```go
package handlers_test

import (
    "bytes"
    "io"
    "mime/multipart"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-chi/chi/v5"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "goimg-datalayer/internal/interfaces/http/handlers"
)

func TestImageHandler_Upload(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name           string
        setup          func(t *testing.T) *multipart.Writer
        setupMocks     func(suite *HandlerTestSuite)
        expectedStatus int
        assertResponse func(t *testing.T, rec *httptest.ResponseRecorder)
    }{
        {
            name: "uploads image successfully",
            setup: func(t *testing.T) *multipart.Writer {
                body := &bytes.Buffer{}
                writer := multipart.NewWriter(body)

                // Add file field
                part, err := writer.CreateFormFile("file", "test.jpg")
                require.NoError(t, err)
                _, err = part.Write(validJPEGData)
                require.NoError(t, err)

                // Add title field
                _ = writer.WriteField("title", "Test Image")

                // Add tags field
                _ = writer.WriteField("tags", "nature,landscape")

                writer.Close()
                return writer
            },
            setupMocks: func(suite *HandlerTestSuite) {
                suite.UploadImageHandler.On("Handle", mock.Anything, mock.MatchedBy(func(cmd commands.UploadImageCommand) bool {
                    return cmd.Title == "Test Image" && len(cmd.Tags) == 2
                })).Return(validImageID, nil)
            },
            expectedStatus: http.StatusCreated,
            assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
                var resp dto.ImageResponse
                err := json.NewDecoder(rec.Body).Decode(&resp)
                require.NoError(t, err)

                assert.NotEmpty(t, resp.ID)
                assert.Equal(t, "Test Image", resp.Title)
                assert.Equal(t, "processing", resp.Status)
            },
        },
        {
            name: "rejects request with no file",
            setup: func(t *testing.T) *multipart.Writer {
                body := &bytes.Buffer{}
                writer := multipart.NewWriter(body)
                writer.Close()
                return writer
            },
            setupMocks:     func(suite *HandlerTestSuite) {},
            expectedStatus: http.StatusBadRequest,
            assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
                var problem dto.ProblemDetails
                json.NewDecoder(rec.Body).Decode(&problem)
                assert.Equal(t, "Validation Error", problem.Title)
                assert.Contains(t, problem.Detail, "file is required")
            },
        },
        {
            name: "enforces rate limiting on upload",
            setup: func(t *testing.T) *multipart.Writer {
                // Setup valid multipart
            },
            setupMocks: func(suite *HandlerTestSuite) {
                // Mock rate limiter returns exceeded
                suite.RateLimiter.On("Allow", mock.Anything, "upload", mock.Anything).
                    Return(false)
            },
            expectedStatus: http.StatusTooManyRequests,
            assertResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
                assert.Equal(t, "50", rec.Header().Get("X-RateLimit-Limit"))
                assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
            },
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            suite := NewHandlerTestSuite(t)
            if tt.setupMocks != nil {
                tt.setupMocks(suite)
            }

            handler := handlers.NewImageHandler(
                suite.UploadImageHandler,
                suite.GetImageHandler,
                suite.RateLimiter,
                &suite.Logger,
            )

            // Create multipart request
            writer := tt.setup(t)
            body := &bytes.Buffer{}
            // ... copy writer content to body

            req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
            req.Header.Set("Content-Type", writer.FormDataContentType())
            req = addAuthContext(req, validUserID) // Add authenticated user

            rec := httptest.NewRecorder()

            handler.Upload(rec, req)

            assert.Equal(t, tt.expectedStatus, rec.Code)
            if tt.assertResponse != nil {
                tt.assertResponse(t, rec)
            }
        })
    }
}
```

### 3.2 Ownership Middleware Test Pattern

```go
func TestOwnershipMiddleware_CheckImageOwnership(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name           string
        imageID        string
        userID         string
        setupMock      func(mockRepo *MockImageRepository)
        expectedStatus int
    }{
        {
            name:    "allows owner to access image",
            imageID: "550e8400-e29b-41d4-a716-446655440000",
            userID:  "7c9e6679-7425-40de-944b-e07fc1f90ae7",
            setupMock: func(mockRepo *MockImageRepository) {
                image := createTestImage(t, imageID, "7c9e6679-7425-40de-944b-e07fc1f90ae7")
                mockRepo.On("FindByID", mock.Anything, mock.Anything).Return(image, nil)
            },
            expectedStatus: http.StatusOK,
        },
        {
            name:    "denies non-owner access",
            imageID: "550e8400-e29b-41d4-a716-446655440000",
            userID:  "different-user-id",
            setupMock: func(mockRepo *MockImageRepository) {
                image := createTestImage(t, imageID, "original-owner-id")
                mockRepo.On("FindByID", mock.Anything, mock.Anything).Return(image, nil)
            },
            expectedStatus: http.StatusForbidden,
        },
        {
            name:    "allows admin access regardless of ownership",
            imageID: "550e8400-e29b-41d4-a716-446655440000",
            userID:  "admin-user-id",
            setupMock: func(mockRepo *MockImageRepository) {
                image := createTestImage(t, imageID, "other-owner-id")
                mockRepo.On("FindByID", mock.Anything, mock.Anything).Return(image, nil)
            },
            expectedStatus: http.StatusOK, // Admin bypasses ownership check
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            mockRepo := new(MockImageRepository)
            tt.setupMock(mockRepo)

            middleware := middleware.NewOwnershipMiddleware(mockRepo)

            // Create test handler
            handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
            })

            wrapped := middleware.CheckImageOwnership(handler)

            req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+tt.imageID, nil)
            req = addAuthContext(req, tt.userID)
            rctx := chi.NewRouteContext()
            rctx.URLParams.Add("imageID", tt.imageID)
            req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

            rec := httptest.NewRecorder()
            wrapped.ServeHTTP(rec, req)

            assert.Equal(t, tt.expectedStatus, rec.Code)
        })
    }
}
```

---

## 4. Integration Testing with Testcontainers

### 4.1 Gallery Repository Tests

Extend the existing repository test patterns for gallery operations:

```go
func TestImageRepository_FindByOwner_WithFilters(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()
    pgContainer, err := containers.NewPostgresContainer(ctx, t)
    require.NoError(t, err)
    defer pgContainer.Terminate(ctx)

    repo := postgres.NewImageRepository(pgContainer.DB)
    ownerID := createTestUser()

    // Create test images with different statuses and visibilities
    publicImage := createTestImageWithVisibility(ownerID, "Public", gallery.VisibilityPublic)
    privateImage := createTestImageWithVisibility(ownerID, "Private", gallery.VisibilityPrivate)
    processingImage := createTestImageWithStatus(ownerID, "Processing", gallery.StatusProcessing)

    require.NoError(t, repo.Save(ctx, publicImage))
    require.NoError(t, repo.Save(ctx, privateImage))
    require.NoError(t, repo.Save(ctx, processingImage))

    tests := []struct {
        name     string
        filter   gallery.ImageFilter
        wantLen  int
    }{
        {
            name: "filters by visibility=public",
            filter: gallery.ImageFilter{
                OwnerID:    ownerID,
                Visibility: ptr(gallery.VisibilityPublic),
            },
            wantLen: 1,
        },
        {
            name: "filters by status=processing",
            filter: gallery.ImageFilter{
                OwnerID: ownerID,
                Status:  ptr(gallery.StatusProcessing),
            },
            wantLen: 1,
        },
        {
            name: "no filters returns all owner's images",
            filter: gallery.ImageFilter{
                OwnerID: ownerID,
            },
            wantLen: 3,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            pagination := shared.DefaultPagination()
            images, total, err := repo.FindByOwner(ctx, tt.filter, pagination)

            require.NoError(t, err)
            assert.Len(t, images, tt.wantLen)
            assert.Equal(t, int64(tt.wantLen), total)
        })
    }
}
```

### 4.2 Full-Text Search Integration Tests

```go
func TestImageRepository_Search_PostgreSQLFullText(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()
    pgContainer, err := containers.NewPostgresContainer(ctx, t)
    require.NoError(t, err)
    defer pgContainer.Terminate(ctx)

    repo := postgres.NewImageRepository(pgContainer.DB)
    ownerID := createTestUser()

    // Create images with searchable titles and descriptions
    testImages := []struct {
        title       string
        description string
        tags        []string
    }{
        {"Sunset Beach", "Beautiful sunset over the ocean", []string{"sunset", "beach", "ocean"}},
        {"Mountain Sunset", "Colorful sunset behind mountains", []string{"sunset", "mountain"}},
        {"City Skyline", "Downtown city at night", []string{"city", "night"}},
    }

    for _, tc := range testImages {
        img := createTestImage(ownerID, tc.title)
        _ = img.UpdateMetadata(tc.title, tc.description)
        _ = img.MarkAsActive()
        _ = img.UpdateVisibility(gallery.VisibilityPublic)

        for _, tagName := range tc.tags {
            tag := gallery.MustNewTag(tagName)
            _ = img.AddTag(tag)
        }

        require.NoError(t, repo.Save(ctx, img))
    }

    tests := []struct {
        name    string
        query   string
        wantLen int
    }{
        {"search by single word in title", "sunset", 2},
        {"search by word in description", "ocean", 1},
        {"search by tag", "mountain", 1},
        {"search with multiple matching", "city", 1},
        {"search with no matches", "nonexistent", 0},
        {"search with partial match", "sun", 2}, // Matches "sunset"
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            searchQuery := gallery.SearchQuery{
                Query:      tt.query,
                Visibility: gallery.VisibilityPublic,
            }

            pagination := shared.DefaultPagination()
            images, total, err := repo.Search(ctx, searchQuery, pagination)

            require.NoError(t, err)
            assert.Len(t, images, tt.wantLen)
            assert.Equal(t, int64(tt.wantLen), total)
        })
    }
}
```

---

## 5. E2E Testing with Newman/Postman

### 5.1 Image Upload Flow

Add to `tests/e2e/postman/goimg-api.postman_collection.json`:

```json
{
  "name": "Image Management",
  "item": [
    {
      "name": "Upload Image",
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test('Status code is 201', function () {",
              "    pm.response.to.have.status(201);",
              "});",
              "",
              "pm.test('Response has image ID', function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData).to.have.property('id');",
              "    pm.expect(jsonData).to.have.property('status');",
              "    pm.expect(jsonData.status).to.eql('processing');",
              "    pm.collectionVariables.set('imageId', jsonData.id);",
              "});",
              "",
              "pm.test('Image has correct metadata', function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData.title).to.eql('E2E Test Image');",
              "    pm.expect(jsonData.tags).to.be.an('array').that.includes('test');",
              "});"
            ]
          }
        }
      ],
      "request": {
        "method": "POST",
        "header": [
          {
            "key": "Authorization",
            "value": "Bearer {{accessToken}}"
          }
        ],
        "body": {
          "mode": "formdata",
          "formdata": [
            {
              "key": "file",
              "type": "file",
              "src": "test-image.jpg"
            },
            {
              "key": "title",
              "value": "E2E Test Image",
              "type": "text"
            },
            {
              "key": "tags",
              "value": "test,e2e",
              "type": "text"
            }
          ]
        },
        "url": {
          "raw": "{{baseUrl}}/api/v1/images",
          "host": ["{{baseUrl}}"],
          "path": ["api", "v1", "images"]
        }
      }
    },
    {
      "name": "Wait for Processing",
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test('Image processed successfully', function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData.status).to.be.oneOf(['active', 'processing']);",
              "    ",
              "    if (jsonData.status === 'processing') {",
              "        // Retry after delay",
              "        setTimeout(function() {}, 2000);",
              "    }",
              "});"
            ]
          }
        }
      ],
      "request": {
        "method": "GET",
        "header": [
          {
            "key": "Authorization",
            "value": "Bearer {{accessToken}}"
          }
        ],
        "url": {
          "raw": "{{baseUrl}}/api/v1/images/{{imageId}}",
          "host": ["{{baseUrl}}"],
          "path": ["api", "v1", "images", "{{imageId}}"]
        }
      }
    },
    {
      "name": "Get Image Variants",
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test('Image has all variants', function () {",
              "    var jsonData = pm.response.json();",
              "    pm.expect(jsonData.variants).to.be.an('array');",
              "    pm.expect(jsonData.variants).to.have.lengthOf(4);",
              "    ",
              "    var variantTypes = jsonData.variants.map(v => v.type);",
              "    pm.expect(variantTypes).to.include.members(['thumbnail', 'small', 'medium', 'large']);",
              "});"
            ]
          }
        }
      ],
      "request": {
        "method": "GET",
        "header": [
          {
            "key": "Authorization",
            "value": "Bearer {{accessToken}}"
          }
        ],
        "url": {
          "raw": "{{baseUrl}}/api/v1/images/{{imageId}}",
          "host": ["{{baseUrl}}"],
          "path": ["api", "v1", "images", "{{imageId}}"]
        }
      }
    }
  ]
}
```

### 5.2 Album CRUD Workflow

```json
{
  "name": "Create Album",
  "event": [
    {
      "listen": "test",
      "script": {
        "exec": [
          "pm.test('Status code is 201', function () {",
          "    pm.response.to.have.status(201);",
          "});",
          "",
          "pm.test('Album created with correct data', function () {",
          "    var jsonData = pm.response.json();",
          "    pm.expect(jsonData.title).to.eql('E2E Test Album');",
          "    pm.expect(jsonData.visibility).to.eql('private');",
          "    pm.collectionVariables.set('albumId', jsonData.id);",
          "});"
        ]
      }
    }
  ],
  "request": {
    "method": "POST",
    "header": [
      {
        "key": "Authorization",
        "value": "Bearer {{accessToken}}"
      }
    ],
    "body": {
      "mode": "raw",
      "raw": "{\n  \"title\": \"E2E Test Album\",\n  \"description\": \"Created by E2E test\",\n  \"visibility\": \"private\"\n}",
      "options": {
        "raw": {
          "language": "json"
        }
      }
    },
    "url": {
      "raw": "{{baseUrl}}/api/v1/albums",
      "host": ["{{baseUrl}}"],
      "path": ["api", "v1", "albums"]
    }
  }
},
{
  "name": "Add Image to Album",
  "event": [
    {
      "listen": "test",
      "script": {
        "exec": [
          "pm.test('Image added to album', function () {",
          "    pm.response.to.have.status(204);",
          "});"
        ]
      }
    }
  ],
  "request": {
    "method": "POST",
    "header": [
      {
        "key": "Authorization",
        "value": "Bearer {{accessToken}}"
      }
    ],
    "body": {
      "mode": "raw",
      "raw": "{\n  \"imageId\": \"{{imageId}}\"\n}",
      "options": {
        "raw": {
          "language": "json"
        }
      }
    },
    "url": {
      "raw": "{{baseUrl}}/api/v1/albums/{{albumId}}/images",
      "host": ["{{baseUrl}}"],
      "path": ["api", "v1", "albums", "{{albumId}}", "images"]
    }
  }
}
```

### 5.3 Search and Filter Tests

```json
{
  "name": "Search Images by Tag",
  "event": [
    {
      "listen": "test",
      "script": {
        "exec": [
          "pm.test('Search returns matching images', function () {",
          "    pm.response.to.have.status(200);",
          "    var jsonData = pm.response.json();",
          "    pm.expect(jsonData.images).to.be.an('array');",
          "    pm.expect(jsonData.total).to.be.above(0);",
          "    ",
          "    // Verify all results have the search tag",
          "    jsonData.images.forEach(function(image) {",
          "        pm.expect(image.tags).to.include('test');",
          "    });",
          "});"
        ]
      }
    }
  ],
  "request": {
    "method": "GET",
    "header": [
      {
        "key": "Authorization",
        "value": "Bearer {{accessToken}}"
      }
    ],
    "url": {
      "raw": "{{baseUrl}}/api/v1/images/search?tags=test&page=1&pageSize=20",
      "host": ["{{baseUrl}}"],
      "path": ["api", "v1", "images", "search"],
      "query": [
        {
          "key": "tags",
          "value": "test"
        },
        {
          "key": "page",
          "value": "1"
        },
        {
          "key": "pageSize",
          "value": "20"
        }
      ]
    }
  }
}
```

---

## 6. Test Coverage Targets

### Sprint 6 Coverage Matrix

| Component | Target | Focus Areas |
|-----------|--------|-------------|
| **Application Commands** | 85% | UploadImage, CreateAlbum, DeleteImage, UpdateImage, AddImageToAlbum, LikeImage, AddComment |
| **Application Queries** | 85% | GetImage, ListImages, SearchImages, GetAlbum, ListAlbums |
| **HTTP Handlers** | 75% | Upload multipart, album CRUD, search endpoints, ownership middleware |
| **Job Handlers** | 80% | ProcessImageJob, error handling, retry logic |
| **Integration** | 70% | Repository operations, search queries, background jobs with Redis |
| **E2E Scenarios** | 100% | Upload→Process→View, Album CRUD, Search, Social features |

### Running Coverage Checks

```bash
# Application layer coverage
go test -coverprofile=app-coverage.out -covermode=atomic ./internal/application/gallery/...
go tool cover -func=app-coverage.out | grep total
# Target: 85%+

# HTTP layer coverage
go test -coverprofile=http-coverage.out -covermode=atomic ./internal/interfaces/http/handlers/...
go tool cover -func=http-coverage.out | grep total
# Target: 75%+

# Overall coverage
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | grep total
# Target: 80%+
```

---

## 7. Testing Challenges and Solutions

### Challenge 1: Async Job Testing Complexity

**Problem**: Testing background jobs with proper timing and status updates.

**Solution**:
- Use mocks for unit tests (immediate feedback)
- Use testcontainers with real Asynq for integration tests
- Add status polling helper for E2E tests
- Test retry logic with transient vs. permanent errors

### Challenge 2: File Upload Testing

**Problem**: Creating valid multipart requests with files.

**Solution**:
- Use `multipart.Writer` in Go tests
- Store test fixtures in `tests/fixtures/images/`
- Use Postman's file upload for E2E tests
- Test boundary conditions (empty files, oversized, invalid formats)

### Challenge 3: Background Processing Timing

**Problem**: Unpredictable timing for variant generation.

**Solution**:
- Mock processor in unit tests for deterministic behavior
- Use status polling with timeout in integration tests
- Add retry logic to E2E tests (wait for `status=active`)
- Test both success and failure paths

### Challenge 4: Search Query Edge Cases

**Problem**: Complex full-text search with filters and pagination.

**Solution**:
- Create comprehensive test data in integration tests
- Test each filter independently
- Test filter combinations
- Verify pagination edge cases (empty, single page, last page)

---

## 8. Test Execution Strategy

### Local Development

```bash
# Fast feedback loop (unit tests only)
make test-unit

# Full test suite with integration
make test

# Specific test
go test -v -run TestUploadImageHandler_Handle ./internal/application/gallery/commands

# Coverage report
make test-coverage
```

### CI Pipeline

```yaml
# .github/workflows/test.yml
jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-unit
      - uses: codecov/codecov-action@v4

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
      redis:
        image: redis:7-alpine
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-integration

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make build
      - run: make run &
      - run: sleep 10  # Wait for server
      - run: make test-e2e
```

---

## 9. Key Testing Patterns Summary

### Existing Patterns to Follow

1. **Table-Driven Tests**: All command/query handlers use table-driven pattern
2. **Mock-Based Application Tests**: Use testify/mock for dependencies
3. **Testcontainers for Integration**: Real PostgreSQL/Redis containers
4. **Parallel Test Execution**: All unit tests use `t.Parallel()`
5. **Test Suite Pattern**: Reusable suite setup with mocked dependencies

### New Patterns for Sprint 6

1. **Multipart Upload Testing**: HTTP handler tests with file uploads
2. **Async Job Testing**: Asynq job handler tests with retry logic
3. **Ownership Middleware Testing**: Authorization checks in HTTP layer
4. **Search Query Testing**: PostgreSQL full-text search integration
5. **E2E Upload Flow**: Newman tests with status polling

---

## 10. Effort Estimation

### Test Implementation Breakdown

| Component | Test Count | Effort | Priority |
|-----------|-----------|--------|----------|
| **UploadImage Command** | 8-10 tests | 4 hours | P0 |
| **Album Commands** | 12-15 tests | 6 hours | P0 |
| **Search Queries** | 8-10 tests | 4 hours | P0 |
| **HTTP Upload Handler** | 6-8 tests | 5 hours | P0 |
| **Ownership Middleware** | 4-6 tests | 3 hours | P0 |
| **ProcessImage Job** | 6-8 tests | 5 hours | P0 |
| **Repository Integration** | 15-20 tests | 8 hours | P1 |
| **E2E Scenarios** | 20-25 requests | 6 hours | P1 |
| **Total** | ~120-150 tests | **41 hours** | |

**Recommended Timeline**: 5-6 days for comprehensive test implementation.

---

## 11. Success Criteria

Sprint 6 testing is complete when:

- [x] Application layer coverage >= 85%
- [x] HTTP handler coverage >= 75%
- [x] Integration tests cover all repository operations
- [x] Async job tests cover success, failure, and retry paths
- [x] E2E tests cover full upload, album, and search workflows
- [x] All tests pass with `-race` detector
- [x] Newman collection includes 30+ test scenarios
- [x] CI pipeline runs all test suites automatically

---

## See Also

- [Test Strategy (General)](./test_strategy.md)
- [Sprint Plan](./sprint_plan.md)
- [Application Layer Guide](../internal/application/CLAUDE.md)
- [HTTP Layer Guide](../internal/interfaces/http/CLAUDE.md)
- [Testing & CI Guide](./testing_ci.md)
