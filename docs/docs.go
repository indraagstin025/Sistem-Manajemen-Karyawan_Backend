// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "API Support",
            "url": "http://www.swagger.io/support",
            "email": "support@swagger.io"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/admin/users": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Mendapatkan semua data users - hanya admin yang dapat mengakses endpoint ini",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Admin"
                ],
                "summary": "Get All Users (Admin Only)",
                "responses": {
                    "200": {
                        "description": "Data users berhasil diambil",
                        "schema": {
                            "$ref": "#/definitions/models.GetAllUsersSuccessResponse"
                        }
                    },
                    "401": {
                        "description": "Tidak terautentikasi",
                        "schema": {
                            "$ref": "#/definitions/models.UnauthorizedErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Akses ditolak - hanya admin",
                        "schema": {
                            "$ref": "#/definitions/models.ForbiddenErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Gagal mengambil data users",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/auth/login": {
            "post": {
                "description": "Melakukan proses login dan mengembalikan token PASETO jika email dan password valid",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Auth"
                ],
                "summary": "Login User",
                "parameters": [
                    {
                        "description": "Login credentials",
                        "name": "credentials",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.UserLoginPayload"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Login berhasil",
                        "schema": {
                            "$ref": "#/definitions/models.LoginSuccessResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid request body",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    },
                    "401": {
                        "description": "Email tidak ditemukan atau password salah",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Gagal menemukan user atau gagal membuat token",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/auth/register": {
            "post": {
                "description": "Mendaftarkan user baru (hanya admin yang dapat melakukan registrasi)",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Auth"
                ],
                "summary": "Register User",
                "parameters": [
                    {
                        "description": "Data registrasi user",
                        "name": "user",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.UserRegisterPayload"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "User berhasil didaftarkan",
                        "schema": {
                            "$ref": "#/definitions/models.RegisterSuccessResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid request body atau validation error",
                        "schema": {
                            "$ref": "#/definitions/models.ValidationErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Gagal hash password atau gagal mendaftarkan user",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/users/change-password": {
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Mengubah password user yang sedang login (required authentication)",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Auth"
                ],
                "summary": "Change Password",
                "parameters": [
                    {
                        "description": "Data untuk mengubah password",
                        "name": "password",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.ChangePasswordPayload"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Password berhasil diubah",
                        "schema": {
                            "$ref": "#/definitions/models.ChangePasswordSuccessResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid request body",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    },
                    "401": {
                        "description": "Tidak terautentikasi atau password lama tidak cocok",
                        "schema": {
                            "$ref": "#/definitions/models.UnauthorizedErrorResponse"
                        }
                    },
                    "500": {
                        "description": "User tidak ditemukan atau gagal update",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/users/{id}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Mendapatkan detail user berdasarkan ID (user hanya bisa melihat data diri sendiri, admin bisa melihat semua)",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Users"
                ],
                "summary": "Get User by ID",
                "parameters": [
                    {
                        "type": "string",
                        "description": "User ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "User berhasil ditemukan",
                        "schema": {
                            "$ref": "#/definitions/models.GetUserSuccessResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid user ID format",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    },
                    "401": {
                        "description": "Tidak terautentikasi",
                        "schema": {
                            "$ref": "#/definitions/models.UnauthorizedErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Akses ditolak - hanya bisa melihat data sendiri",
                        "schema": {
                            "$ref": "#/definitions/models.ForbiddenErrorResponse"
                        }
                    },
                    "404": {
                        "description": "User tidak ditemukan",
                        "schema": {
                            "$ref": "#/definitions/models.NotFoundErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Gagal mengambil data user",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    }
                }
            },
            "put": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Update data user (user hanya bisa update data diri sendiri, admin bisa update semua)",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Users"
                ],
                "summary": "Update User",
                "parameters": [
                    {
                        "type": "string",
                        "description": "User ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Data update user",
                        "name": "user",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.UserUpdatePayload"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "User berhasil diupdate",
                        "schema": {
                            "$ref": "#/definitions/models.UpdateUserSuccessResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid request body atau user ID",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    },
                    "401": {
                        "description": "Tidak terautentikasi",
                        "schema": {
                            "$ref": "#/definitions/models.UnauthorizedErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Akses ditolak - hanya bisa update data sendiri",
                        "schema": {
                            "$ref": "#/definitions/models.ForbiddenErrorResponse"
                        }
                    },
                    "404": {
                        "description": "User tidak ditemukan",
                        "schema": {
                            "$ref": "#/definitions/models.NotFoundErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Gagal update user",
                        "schema": {
                            "$ref": "#/definitions/models.ErrorResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "models.ChangePasswordPayload": {
            "type": "object",
            "required": [
                "new_password",
                "old_password"
            ],
            "properties": {
                "new_password": {
                    "type": "string",
                    "maxLength": 50,
                    "minLength": 8
                },
                "old_password": {
                    "type": "string"
                }
            }
        },
        "models.ChangePasswordSuccessResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "Password berhasil diubah."
                }
            }
        },
        "models.ErrorResponse": {
            "type": "object",
            "properties": {
                "details": {
                    "type": "string",
                    "example": "validation failed"
                },
                "error": {
                    "type": "string",
                    "example": "Invalid request body"
                }
            }
        },
        "models.ForbiddenErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "Akses ditolak. Hanya admin yang dapat mengakses endpoint ini"
                }
            }
        },
        "models.GetAllUsersSuccessResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "Data users berhasil diambil"
                },
                "total": {
                    "type": "integer",
                    "example": 10
                },
                "users": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/models.User"
                    }
                }
            }
        },
        "models.GetUserSuccessResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "User berhasil ditemukan"
                },
                "user": {
                    "$ref": "#/definitions/models.User"
                }
            }
        },
        "models.LoginSuccessResponse": {
            "type": "object",
            "properties": {
                "is_first_login": {
                    "type": "boolean",
                    "example": true
                },
                "message": {
                    "type": "string",
                    "example": "Login berhasil"
                },
                "role": {
                    "type": "string",
                    "example": "karyawan"
                },
                "token": {
                    "type": "string",
                    "example": "v2.local.Ft9QcxZhJXEYyb7-bMM..."
                },
                "user_id": {
                    "type": "string",
                    "example": "507f1f77bcf86cd799439011"
                }
            }
        },
        "models.NotFoundErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "User tidak ditemukan"
                }
            }
        },
        "models.RegisterSuccessResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "User berhasil didaftarkan (oleh admin)"
                },
                "user_id": {
                    "type": "string",
                    "example": "507f1f77bcf86cd799439011"
                }
            }
        },
        "models.UnauthorizedErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "Token tidak valid atau tidak ada"
                }
            }
        },
        "models.UpdateUserSuccessResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "User berhasil diupdate"
                },
                "user_id": {
                    "type": "string",
                    "example": "507f1f77bcf86cd799439011"
                }
            }
        },
        "models.User": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "string"
                },
                "base_salary": {
                    "type": "number"
                },
                "created_at": {
                    "type": "string"
                },
                "department": {
                    "type": "string"
                },
                "email": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "is_first_login": {
                    "type": "boolean"
                },
                "name": {
                    "type": "string"
                },
                "password": {
                    "type": "string"
                },
                "photo": {
                    "type": "string"
                },
                "position": {
                    "type": "string"
                },
                "role": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "models.UserLoginPayload": {
            "type": "object",
            "required": [
                "email",
                "password"
            ],
            "properties": {
                "email": {
                    "type": "string"
                },
                "password": {
                    "type": "string"
                }
            }
        },
        "models.UserRegisterPayload": {
            "type": "object",
            "required": [
                "name",
                "password",
                "role"
            ],
            "properties": {
                "address": {
                    "type": "string",
                    "maxLength": 255,
                    "minLength": 5
                },
                "base_salary": {
                    "type": "number",
                    "minimum": 0
                },
                "department": {
                    "type": "string"
                },
                "email": {
                    "type": "string"
                },
                "name": {
                    "type": "string",
                    "maxLength": 100,
                    "minLength": 3
                },
                "password": {
                    "type": "string",
                    "maxLength": 50,
                    "minLength": 8
                },
                "photo": {
                    "type": "string"
                },
                "position": {
                    "type": "string"
                },
                "role": {
                    "type": "string",
                    "enum": [
                        "admin",
                        "karyawan"
                    ]
                }
            }
        },
        "models.UserUpdatePayload": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "string",
                    "maxLength": 255,
                    "minLength": 5
                },
                "base_salary": {
                    "type": "number",
                    "minimum": 0
                },
                "department": {
                    "type": "string"
                },
                "email": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "photo": {
                    "type": "string"
                },
                "position": {
                    "type": "string"
                }
            }
        },
        "models.ValidationErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "Validation failed"
                },
                "errors": {
                    "type": "string",
                    "example": "email: email tidak valid, password: password terlalu pendek"
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "description": "Type \"Bearer\" followed by a space and JWT token.",
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:3000",
	BasePath:         "/api/v1",
	Schemes:          []string{"http"},
	Title:            "Sistem Manajemen Karyawan API",
	Description:      "API untuk sistem manajemen karyawan dengan authentication dan authorization",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
