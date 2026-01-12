package community

// GroupSettings is a value object containing configuration options for a group.
// Settings control member permissions and group behavior.
type GroupSettings struct {
	requireApproval    bool // Images require admin approval before appearing in group
	allowMemberInvites bool // Members can invite others (not just admins)
	allowMemberAlbums  bool // Members can create group albums
	maxMembers         int  // Maximum member capacity (0 = unlimited)
}

// NewGroupSettings creates a new GroupSettings value object with validation.
func NewGroupSettings(requireApproval, allowMemberInvites, allowMemberAlbums bool, maxMembers int) (GroupSettings, error) {
	if maxMembers < 0 {
		return GroupSettings{}, ErrInvalidMaxMembers
	}

	return GroupSettings{
		requireApproval:    requireApproval,
		allowMemberInvites: allowMemberInvites,
		allowMemberAlbums:  allowMemberAlbums,
		maxMembers:         maxMembers,
	}, nil
}

// DefaultGroupSettings returns the default settings for new groups.
// Defaults: no approval required, members can invite, members can create albums, unlimited members.
func DefaultGroupSettings() GroupSettings {
	return GroupSettings{
		requireApproval:    false,
		allowMemberInvites: true,
		allowMemberAlbums:  true,
		maxMembers:         0, // Unlimited
	}
}

// RequireApproval returns true if images require admin approval.
func (s GroupSettings) RequireApproval() bool {
	return s.requireApproval
}

// AllowMemberInvites returns true if members can invite others.
func (s GroupSettings) AllowMemberInvites() bool {
	return s.allowMemberInvites
}

// AllowMemberAlbums returns true if members can create group albums.
func (s GroupSettings) AllowMemberAlbums() bool {
	return s.allowMemberAlbums
}

// MaxMembers returns the maximum member capacity (0 = unlimited).
func (s GroupSettings) MaxMembers() int {
	return s.maxMembers
}

// HasMemberLimit returns true if the group has a member limit.
func (s GroupSettings) HasMemberLimit() bool {
	return s.maxMembers > 0
}

// Equals returns true if this GroupSettings equals the other GroupSettings.
func (s GroupSettings) Equals(other GroupSettings) bool {
	return s.requireApproval == other.requireApproval &&
		s.allowMemberInvites == other.allowMemberInvites &&
		s.allowMemberAlbums == other.allowMemberAlbums &&
		s.maxMembers == other.maxMembers
}

// WithRequireApproval returns a new GroupSettings with the requireApproval setting changed.
func (s GroupSettings) WithRequireApproval(require bool) GroupSettings {
	return GroupSettings{
		requireApproval:    require,
		allowMemberInvites: s.allowMemberInvites,
		allowMemberAlbums:  s.allowMemberAlbums,
		maxMembers:         s.maxMembers,
	}
}

// WithAllowMemberInvites returns a new GroupSettings with the allowMemberInvites setting changed.
func (s GroupSettings) WithAllowMemberInvites(allow bool) GroupSettings {
	return GroupSettings{
		requireApproval:    s.requireApproval,
		allowMemberInvites: allow,
		allowMemberAlbums:  s.allowMemberAlbums,
		maxMembers:         s.maxMembers,
	}
}

// WithAllowMemberAlbums returns a new GroupSettings with the allowMemberAlbums setting changed.
func (s GroupSettings) WithAllowMemberAlbums(allow bool) GroupSettings {
	return GroupSettings{
		requireApproval:    s.requireApproval,
		allowMemberInvites: s.allowMemberInvites,
		allowMemberAlbums:  allow,
		maxMembers:         s.maxMembers,
	}
}

// WithMaxMembers returns a new GroupSettings with the maxMembers setting changed.
func (s GroupSettings) WithMaxMembers(max int) (GroupSettings, error) {
	if max < 0 {
		return GroupSettings{}, ErrInvalidMaxMembers
	}
	return GroupSettings{
		requireApproval:    s.requireApproval,
		allowMemberInvites: s.allowMemberInvites,
		allowMemberAlbums:  s.allowMemberAlbums,
		maxMembers:         max,
	}, nil
}
