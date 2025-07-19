package models

type RegisterSuccessResponse struct {
    Message string `json:"message" example:"User berhasil didaftarkan (oleh admin)"`
    UserID  string `json:"user_id" example:"507f1f77bcf86cd799439011"`
}

type LoginSuccessResponse struct {
    Message       string `json:"message" example:"Login berhasil"`
    Token         string `json:"token" example:"v2.local.Ft9QcxZhJXEYyb7-bMM..."`
    UserID        string `json:"user_id" example:"507f1f77bcf86cd799439011"`
    Role          string `json:"role" example:"karyawan"`
    IsFirstLogin  bool   `json:"is_first_login" example:"true"`
}

type ChangePasswordSuccessResponse struct {
    Message string `json:"message" example:"Password berhasil diubah."`
}

type GetUserSuccessResponse struct {
    Message string `json:"message" example:"User berhasil ditemukan"`
    User    User   `json:"user"`
}

type GetAllUsersSuccessResponse struct {
    Message string `json:"message" example:"Data users berhasil diambil"`
    Users   []User `json:"users"`
    Total   int    `json:"total" example:"10"`
}

type UpdateUserSuccessResponse struct {
    Message string `json:"message" example:"User berhasil diupdate"`
    UserID  string `json:"user_id" example:"507f1f77bcf86cd799439011"`
}

type DeleteUserSuccessResponse struct {
    Message string `json:"message" example:"User berhasil dihapus"`
    UserID  string `json:"user_id" example:"507f1f77bcf86cd799439011"`
}

type ErrorResponse struct {
    Error   string `json:"error" example:"Invalid request body"`
    Details string `json:"details,omitempty" example:"validation failed"`
}

type ValidationErrorResponse struct {
    Error  string `json:"error" example:"Validation failed"`
    Errors string `json:"errors" example:"email: email tidak valid, password: password terlalu pendek"`
}

type UnauthorizedErrorResponse struct {
    Error string `json:"error" example:"Token tidak valid atau tidak ada"`
}

type ForbiddenErrorResponse struct {
    Error string `json:"error" example:"Akses ditolak. Hanya admin yang dapat mengakses endpoint ini"`
}

type NotFoundErrorResponse struct {
    Error string `json:"error" example:"User tidak ditemukan"`
}

type LogoutSuccessResponse struct {
	Message string `json:"message" example:"Logout berhasil. Silakan hapus token dari sisi client."`
}
