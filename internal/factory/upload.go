package factory

import (
	"fmt"

	"app/internal/db"
)

// Upload creates the database record of an uploaded image. No file is written:
// use the upload endpoint when a test needs the bytes on disk.
func (f *Factory) Upload(userID int32) *db.Upload {
	f.t.Helper()

	upload, err := f.queries.FactoryCreateUpload(ctx, db.FactoryCreateUploadParams{
		UserID:       userID,
		RelativePath: fmt.Sprintf("%d/test_%s.jpg", userID, unique()),
	})
	if err != nil {
		f.t.Fatalf("factory.Upload: %v", err)
	}
	return &upload
}
