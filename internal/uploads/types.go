package uploads

import (
	"app/internal"
)

// UploadResponse represents upload information
type UploadResponse struct {
	ID               int32  `json:"id"`
	UserID           int32  `json:"user_id"`
	FolderID         int32  `json:"folder_id"`
	Type             string `json:"type" enums:"image,video,audio,document,other"`
	RelativePath     string `json:"relative_path"`
	FullURL          string `json:"full_url"`
	OriginalFilename string `json:"original_filename"`
	FileSize         int64  `json:"file_size"`
	MimeType         string `json:"mime_type"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// UploadDataResponse wraps upload data in response
type UploadDataResponse struct {
	Data *UploadResponse `json:"data"`
}

// PaginatedUploadsResponse wraps paginated uploads in response
type PaginatedUploadsResponse struct {
	Data       []UploadResponse        `json:"data"`
	Pagination internal.PaginationMeta `json:"pagination"`
}
