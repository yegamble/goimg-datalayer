-- +goose Up
-- Moderation context tables: reports, user_bans, reviews

-- Reports table - abuse and content violation reports
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    reason VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ,
    resolution TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT reports_reason_check CHECK (reason IN ('spam', 'inappropriate', 'copyright', 'harassment', 'other')),
    CONSTRAINT reports_status_check CHECK (status IN ('pending', 'reviewing', 'resolved', 'dismissed')),
    CONSTRAINT reports_description_not_empty CHECK (LENGTH(TRIM(description)) > 0),
    CONSTRAINT reports_description_length CHECK (LENGTH(description) <= 1000),
    CONSTRAINT reports_resolution_length CHECK (LENGTH(resolution) <= 1000),
    CONSTRAINT reports_resolved_consistency CHECK (
        (status IN ('resolved', 'dismissed') AND resolved_by IS NOT NULL AND resolved_at IS NOT NULL) OR
        (status IN ('pending', 'reviewing') AND resolved_by IS NULL AND resolved_at IS NULL)
    )
);

-- Indexes for reports
CREATE INDEX idx_reports_reporter_id ON reports(reporter_id);
CREATE INDEX idx_reports_image_id ON reports(image_id);
CREATE INDEX idx_reports_status ON reports(status);
CREATE INDEX idx_reports_created_at ON reports(created_at DESC);
CREATE INDEX idx_reports_resolved_by ON reports(resolved_by) WHERE resolved_by IS NOT NULL;
CREATE INDEX idx_reports_pending ON reports(created_at DESC) WHERE status = 'pending';
CREATE INDEX idx_reports_reviewing ON reports(created_at DESC) WHERE status = 'reviewing';

COMMENT ON TABLE reports IS 'Content abuse and violation reports';
COMMENT ON COLUMN reports.reason IS 'Report reason: spam, inappropriate, copyright, harassment, other';
COMMENT ON COLUMN reports.status IS 'Report status: pending, reviewing, resolved, dismissed';
COMMENT ON COLUMN reports.description IS 'User-provided description of the issue (max 1000 chars)';
COMMENT ON COLUMN reports.resolution IS 'Moderator resolution notes (max 1000 chars)';
COMMENT ON COLUMN reports.resolved_by IS 'Moderator who resolved the report (NULL if unresolved)';
COMMENT ON COLUMN reports.resolved_at IS 'Timestamp when report was resolved (NULL if unresolved)';

-- User bans table - temporary and permanent user restrictions
CREATE TABLE user_bans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    banned_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT bans_reason_not_empty CHECK (LENGTH(TRIM(reason)) > 0),
    CONSTRAINT bans_reason_length CHECK (LENGTH(reason) <= 500),
    CONSTRAINT bans_revocation_consistency CHECK (
        (revoked_at IS NOT NULL AND revoked_by IS NOT NULL) OR
        (revoked_at IS NULL AND revoked_by IS NULL)
    ),
    CONSTRAINT bans_expiry_in_future CHECK (expires_at IS NULL OR expires_at > created_at)
);

-- Indexes for user_bans
CREATE INDEX idx_bans_user_id ON user_bans(user_id);
CREATE INDEX idx_bans_banned_by ON user_bans(banned_by);
CREATE INDEX idx_bans_created_at ON user_bans(created_at DESC);
CREATE INDEX idx_bans_expires_at ON user_bans(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_bans_active ON user_bans(user_id, created_at DESC) WHERE revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW());
CREATE INDEX idx_bans_revoked_by ON user_bans(revoked_by) WHERE revoked_by IS NOT NULL;

COMMENT ON TABLE user_bans IS 'User bans (temporary and permanent)';
COMMENT ON COLUMN user_bans.user_id IS 'Banned user';
COMMENT ON COLUMN user_bans.banned_by IS 'Moderator who issued the ban';
COMMENT ON COLUMN user_bans.reason IS 'Ban reason (max 500 chars)';
COMMENT ON COLUMN user_bans.expires_at IS 'Expiration timestamp (NULL for permanent bans)';
COMMENT ON COLUMN user_bans.revoked_at IS 'Timestamp when ban was manually revoked (NULL if not revoked)';
COMMENT ON COLUMN user_bans.revoked_by IS 'Moderator who revoked the ban (NULL if not revoked)';

-- Reviews table - immutable audit trail of moderation decisions
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(20) NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT reviews_action_check CHECK (action IN ('dismiss', 'warn', 'remove', 'ban')),
    CONSTRAINT reviews_notes_length CHECK (LENGTH(notes) <= 2000)
);

-- Indexes for reviews
CREATE INDEX idx_reviews_report_id ON reviews(report_id);
CREATE INDEX idx_reviews_reviewer_id ON reviews(reviewer_id);
CREATE INDEX idx_reviews_created_at ON reviews(created_at DESC);
CREATE INDEX idx_reviews_action ON reviews(action);

COMMENT ON TABLE reviews IS 'Immutable audit trail of moderation decisions';
COMMENT ON COLUMN reviews.report_id IS 'Report being reviewed';
COMMENT ON COLUMN reviews.reviewer_id IS 'Moderator who performed the review';
COMMENT ON COLUMN reviews.action IS 'Action taken: dismiss, warn, remove, ban';
COMMENT ON COLUMN reviews.notes IS 'Reviewer notes (max 2000 chars)';

-- +goose Down
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS user_bans;
DROP TABLE IF EXISTS reports;
