package models

// Success Response Models

// RegisterSuccessResponse represents successful registration response
type RegisterSuccessResponse struct {
    Message string `json:"message" example:"User berhasil didaftarkan (oleh admin)"`
    UserID  string `json:"user_id" example:"507f1f77bcf86cd799439011"`
}

// LoginSuccessResponse represents successful login response
type LoginSuccessResponse struct {
    Message       string `json:"message" example:"Login berhasil"`
    Token         string `json:"token" example:"v2.local.Ft9QcxZhJXEYyb7-bMM..."`
    UserID        string `json:"user_id" example:"507f1f77bcf86cd799439011"`
    Role          string `json:"role" example:"karyawan"`
    IsFirstLogin  bool   `json:"is_first_login" example:"true"`
}

// ChangePasswordSuccessResponse represents successful password change response
type ChangePasswordSuccessResponse struct {
    Message string `json:"message" example:"Password berhasil diubah."`
}

// GetUserSuccessResponse represents successful get user response
type GetUserSuccessResponse struct {
    Message string `json:"message" example:"User berhasil ditemukan"`
    User    User   `json:"user"`
}

// GetAllUsersSuccessResponse represents successful get all users response
type GetAllUsersSuccessResponse struct {
    Message string `json:"message" example:"Data users berhasil diambil"`
    Users   []User `json:"users"`
    Total   int    `json:"total" example:"10"`
}

// UpdateUserSuccessResponse represents successful update user response
type UpdateUserSuccessResponse struct {
    Message string `json:"message" example:"User berhasil diupdate"`
    UserID  string `json:"user_id" example:"507f1f77bcf86cd799439011"`
}

// DeleteUserSuccessResponse represents successful delete user response
type DeleteUserSuccessResponse struct {
    Message string `json:"message" example:"User berhasil dihapus"`
    UserID  string `json:"user_id" example:"507f1f77bcf86cd799439011"`
}

// Error Response Models

// ErrorResponse represents basic error response structure
type ErrorResponse struct {
    Error   string `json:"error" example:"Invalid request body"`
    Details string `json:"details,omitempty" example:"validation failed"`
}

// ValidationErrorResponse represents validation error response
type ValidationErrorResponse struct {
    Error  string `json:"error" example:"Validation failed"`
    Errors string `json:"errors" example:"email: email tidak valid, password: password terlalu pendek"`
}

// UnauthorizedErrorResponse represents unauthorized error response
type UnauthorizedErrorResponse struct {
    Error string `json:"error" example:"Token tidak valid atau tidak ada"`
}

// ForbiddenErrorResponse represents forbidden error response
type ForbiddenErrorResponse struct {
    Error string `json:"error" example:"Akses ditolak. Hanya admin yang dapat mengakses endpoint ini"`
}

// NotFoundErrorResponse represents not found error response
type NotFoundErrorResponse struct {
    Error string `json:"error" example:"User tidak ditemukan"`
}