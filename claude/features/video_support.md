# Feature Specification: Video Support (Sprint 21)

**Status:** DRAFT
**Owner:** Product Architect
**Type:** New Feature

## Overview
To evolve into a "PeerTube-like" platform, the system must support video uploads, processing, and streaming. This specification outlines the architecture for handling video content, moving beyond simple image hosting.

## User Stories
*   **U1:** As a user, I want to upload video files (MP4, WebM, MOV) so I can share them.
*   **U2:** As a viewer, I want to watch videos with adaptive streaming (HLS) so they play smoothly on my connection.
*   **U3:** As a system, I want to generate thumbnails and previews for videos automatically.
*   **U4:** As a moderator, I want to scan video frames for NSFW content.

## Technical Architecture

### 1. Upload Flow
*   **Endpoint:** `POST /api/v1/videos` (or expand `POST /api/v1/images`)
    *   *Decision:* Use a separate `Video` entity or expand `Image` to `Media`?
    *   *Recommendation:* Use a separate `Video` entity to handle the complexity of transcoding states and multiple files (segments).
*   **Tus Protocol:** Implement resumable uploads using `tusd` or a Go implementation. Video files are large; standard multipart/form-data is insufficient.

### 2. Processing Pipeline (Asynq)
*   **Job:** `process_video_upload`
*   **Tool:** FFmpeg (via `u2takey/ffmpeg-go` or direct exec).
*   **Steps:**
    1.  **Probe:** Check codec, resolution, duration.
    2.  **Thumbnail:** Extract frame at 10% or middle duration.
    3.  **Transcode:** Convert to HLS (HTTP Live Streaming).
        *   Master playlist (`master.m3u8`).
        *   Variant playlists (1080p, 720p, 480p).
        *   TS segments.
    4.  **Storage:** Upload all segments to S3/IPFS.

### 3. Data Model

#### `videos` Table
```sql
CREATE TABLE videos (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(255),
    description TEXT,

    -- File Info
    original_filename VARCHAR(255),
    size_bytes BIGINT,
    duration_seconds INTEGER,
    mime_type VARCHAR(100),

    -- Processing
    status VARCHAR(50) DEFAULT 'pending', -- pending, processing, ready, failed
    processing_error TEXT,

    -- Storage
    storage_path VARCHAR(255), -- Prefix for HLS files
    thumbnail_path VARCHAR(255),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 4. API Endpoints

*   `POST /api/v1/videos` - Initiate upload (Tus or Multipart).
*   `GET /api/v1/videos/{id}` - Get video metadata + playback URL.
*   `GET /api/v1/videos/{id}/manifest.m3u8` - Serve HLS manifest (proxy to S3 or signed URL).

### 5. Storage Strategy
*   **Structure:** `/videos/{uuid}/{resolution}/`
*   **HLS:**
    *   `/videos/{uuid}/master.m3u8`
    *   `/videos/{uuid}/1080p/playlist.m3u8`
    *   `/videos/{uuid}/1080p/segment_001.ts`

### 6. Security & Moderation
*   **Scanning:** Extract keyframes (e.g., every 5 seconds) and send to NSFW scanner.
*   **Quotas:** Videos consume significant storage. Enforce strict limits based on user tier (Sprint 22).

## Plan of Attack (Phases)

### Phase 1: MVP Video
*   Direct upload (no Tus yet).
*   MP4 passthrough (no HLS).
*   Simple playback tag `<video src="...">`.

### Phase 2: Transcoding
*   Background FFmpeg worker.
*   HLS generation.
*   Thumbnail generation.

### Phase 3: Advanced
*   Tus resumable uploads.
*   Adaptive bitrate streaming.
*   NSFW frame scanning.

## Dependencies
*   FFmpeg installed in Docker container / Worker environment.
*   Storage quota management.
