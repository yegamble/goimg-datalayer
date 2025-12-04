package postgres

// This file exists solely for compile-time verification that repositories implement their interfaces.
// These variables will never be instantiated at runtime.

import (
	"github.com/yegamble/goimg-datalayer/internal/domain/gallery"
)

// Compile-time interface implementation checks.
var (
	_ gallery.ImageRepository = (*ImageRepository)(nil)
	_ gallery.AlbumRepository = (*AlbumRepository)(nil)
)
