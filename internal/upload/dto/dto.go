package dto

type UploadResponse struct {
	URL string `json:"url" example:"/uploads/8db9f3e2-8a45-4a70-b3d1-167b4f97e121.jpg"`
}

// Имя типа не ErrorResponse: одноимённые схемы из разных пакетов swag разводит
// нечитаемыми именами — тот же случай, что с ItemError.
type UploadError struct {
	Error string `json:"error"`
}
