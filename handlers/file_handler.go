package handlers

import (
	// Pastikan path import ini benar sesuai struktur folder Anda
	"Sistem-Manajemen-Karyawan/config"
	"bytes" // Import paket bytes
	"io"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FileHandler adalah struct BARU yang kita buat untuk menangani logika file.
type FileHandler struct {
	// Kosong untuk saat ini, tapi bisa ditambahkan dependensi jika perlu.
}

// NewFileHandler adalah "konstruktor" untuk membuat FileHandler.
func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

// GetFileFromGridFS adalah fungsi yang akan mengambil file dari database dan mengirimkannya.
func (h *FileHandler) GetFileFromGridFS(c *fiber.Ctx) error {
	// Ambil ID file dari URL
	fileIDHex := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(fileIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format File ID tidak valid"})
	}

	// Dapatkan koneksi ke GridFS
	bucket, err := config.GetGridFSBucket()
	if err != nil {
		log.Printf("ERROR: Gagal mendapatkan bucket GridFS: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengakses penyimpanan file"})
	}

	// Buka file dari database untuk di-download
	downloadStream, err := bucket.OpenDownloadStream(objectID)
	if err != nil {
		log.Printf("ERROR: File tidak ditemukan dengan ID %s: %v", fileIDHex, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File tidak ditemukan"})
	}
	defer downloadStream.Close()

	// === PERBAIKAN DI SINI: Baca seluruh file ke buffer di memori ===
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, downloadStream); err != nil {
		log.Printf("ERROR: Gagal membaca file dari GridFS ke buffer: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membaca data file"})
	}
	// ===============================================================

	// Ambil metadata file (seperti nama file asli) dari database
	fileInfo := downloadStream.GetFile()

	// Deteksi tipe konten dari buffer
	contentType := http.DetectContentType(buf.Bytes())
	c.Set("Content-Type", contentType)

	// Gunakan nama file asli dari database
	c.Set("Content-Disposition", "inline; filename=\""+fileInfo.Name+"\"")

	// Kirim buffer yang berisi seluruh file ke browser
	return c.Send(buf.Bytes())
}
