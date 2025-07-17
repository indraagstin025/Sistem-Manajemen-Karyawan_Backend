package handlers

import (
	"Sistem-Manajemen-Karyawan/config"
	"bytes" 
	"io"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)


type FileHandler struct {
	
}

// NewFileHandler adalah "konstruktor" untuk membuat FileHandler.
func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

// GetFileFromGridFS godoc
// @Summary Get File from GridFS by ID
// @Description Mengambil file dari GridFS berdasarkan file ID dan mengirimkannya sebagai response
// @Tags Files
// @Accept json
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path string true "File ID"
// @Success 200 {file} file "File berhasil diambil"
// @Failure 400 {object} object{error=string} "Format File ID tidak valid"
// @Failure 404 {object} object{error=string} "File tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal mengakses atau membaca file"
// @Router /files/{id} [get]
func (h *FileHandler) GetFileFromGridFS(c *fiber.Ctx) error {
	fileIDHex := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(fileIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format File ID tidak valid"})
	}

	bucket, err := config.GetGridFSBucket()
	if err != nil {
		log.Printf("ERROR: Gagal mendapatkan bucket GridFS: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengakses penyimpanan file"})
	}

	downloadStream, err := bucket.OpenDownloadStream(objectID)
	if err != nil {
		log.Printf("ERROR: File tidak ditemukan dengan ID %s: %v", fileIDHex, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File tidak ditemukan"})
	}
	defer downloadStream.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, downloadStream); err != nil {
		log.Printf("ERROR: Gagal membaca file dari GridFS ke buffer: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membaca data file"})
	}
	
	fileInfo := downloadStream.GetFile()

	contentType := http.DetectContentType(buf.Bytes())
	c.Set("Content-Type", contentType)

	c.Set("Content-Disposition", "inline; filename=\""+fileInfo.Name+"\"")

	return c.Send(buf.Bytes())
}


// GetFileByFilename godoc
// @Summary Get File from GridFS by Filename
// @Description Mengambil file dari GridFS berdasarkan nama file dan mengirimkannya sebagai response
// @Tags Files
// @Accept json
// @Produce application/octet-stream
// @Security BearerAuth
// @Param filename path string true "Filename"
// @Success 200 {file} file "File berhasil diambil"
// @Failure 400 {object} object{error=string} "Nama file tidak boleh kosong"
// @Failure 404 {object} object{error=string} "File tidak ditemukan"
// @Failure 500 {object} object{error=string} "Gagal mengakses atau membaca file"
// @Router /files/by-name/{filename} [get]
func (h *FileHandler) GetFileByFilename(c *fiber.Ctx) error {
	filename := c.Params("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama file tidak boleh kosong"})
	}

	bucket, err := config.GetGridFSBucket()
	if err != nil {
		log.Printf("ERROR: Gagal mendapatkan bucket GridFS: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengakses penyimpanan file"})
	}

	downloadStream, err := bucket.OpenDownloadStreamByName(filename)
	if err != nil {
		log.Printf("ERROR: File tidak ditemukan dengan nama %s: %v", filename, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File tidak ditemukan"})
	}
	defer downloadStream.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, downloadStream); err != nil {
		log.Printf("ERROR: Gagal membaca file dari GridFS ke buffer: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membaca data file"})
	}

	fileInfo := downloadStream.GetFile()
	contentType := http.DetectContentType(buf.Bytes())
	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", "inline; filename=\""+fileInfo.Name+"\"")

	return c.Send(buf.Bytes())
}