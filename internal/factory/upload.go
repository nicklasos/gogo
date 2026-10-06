package factory

import (
	"fmt"

	"app/internal/db"
)

// Upload creates the database record of an uploaded image. No file is written:
// use the upload endpoint when a test needs the bytes on disk.
func Upload(t TB, conn DB, userID int32) *db.Upload {
	t.Helper()

	var upload db.Upload
	err := conn.QueryRow(ctx,
		`INSERT INTO uploads (user_id, folder_id, type, relative_path, original_filename, file_size, mime_type)
		 VALUES ($1, $1, 'image', $2, 'test.jpg', 1024, 'image/jpeg')
		 RETURNING id, user_id, folder_id, type, relative_path, original_filename, file_size, mime_type, created_at, updated_at`,
		userID, fmt.Sprintf("%d/test_%s.jpg", userID, unique()),
	).Scan(&upload.ID, &upload.UserID, &upload.FolderID, &upload.Type, &upload.RelativePath, &upload.OriginalFilename,
		&upload.FileSize, &upload.MimeType, &upload.CreatedAt, &upload.UpdatedAt)
	if err != nil {
		t.Fatalf("factory.Upload: %v", err)
	}
	return &upload
}
