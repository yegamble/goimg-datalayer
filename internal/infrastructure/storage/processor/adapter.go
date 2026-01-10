//go:build cgo

package processor

import (
	"context"

	"github.com/yegamble/goimg-datalayer/internal/application/gallery/commands"
)

// ProcessorAdapter adapts the Processor to implement the commands.ImageProcessor interface.
// This allows the infrastructure layer processor to be used by the application layer
// without creating a direct dependency on infrastructure types.
type ProcessorAdapter struct {
	processor *Processor
}

// NewProcessorAdapter creates a new adapter wrapping the given processor.
func NewProcessorAdapter(processor *Processor) *ProcessorAdapter {
	return &ProcessorAdapter{processor: processor}
}

// GenerateCustomVariant implements commands.ImageProcessor.
// It delegates to the underlying processor and converts the result to application layer types.
func (a *ProcessorAdapter) GenerateCustomVariant(
	ctx context.Context,
	input []byte,
	maxWidth, maxHeight int,
	format string,
	quality int,
	cropMode string,
) (*commands.CustomVariantData, error) {
	result, err := a.processor.GenerateCustomVariant(
		ctx, input, maxWidth, maxHeight, format, quality, cropMode,
	)
	if err != nil {
		return nil, err
	}

	// Convert processor.CustomVariantData to commands.CustomVariantData
	return &commands.CustomVariantData{
		Data:        result.Data,
		Width:       result.Width,
		Height:      result.Height,
		Format:      result.Format,
		ContentType: result.ContentType,
		FileSize:    result.FileSize,
	}, nil
}

// Ensure ProcessorAdapter implements the commands.ImageProcessor interface.
var _ commands.ImageProcessor = (*ProcessorAdapter)(nil)
