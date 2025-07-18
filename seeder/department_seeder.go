// file: seeder/department_seeder.go

package seeder

import (
	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/repository"
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SeedDepartments berfungsi untuk memasukkan data departemen dummy ke database
func SeedDepartments(departmentRepo repository.DepartmentRepository) {
	log.Println("🌱 Memulai seeding departemen...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Pastikan daftar nama departemen ini adalah yang benar-benar akan di-seed
	// Perhatikan penulisan (kapitalisasi, spasi, tanda baca)
	departmentsData := []string{
		"Keuangan",
		"Sumber Daya Manusia (HRD)",
		"Teknologi Informasi (IT)",
		"Pemasaran",
		"Penjualan",
		"Produksi",
		"Riset & Pengembangan (RnD)",
		"Layanan Pelanggan",
		"Logistik",
	}

	for _, deptName := range departmentsData {
		existingDept, err := departmentRepo.FindDepartmentByName(ctx, deptName)
		// Perbaikan kecil: Cek err == nil DAN existingDept != nil untuk memastikan departemen ditemukan
		if err == nil && existingDept != nil && existingDept.Name == deptName {
			fmt.Printf("Skipping: Departemen '%s' sudah ada.\n", deptName)
			continue
		}

		newDepartment := &models.Department{
			ID:        primitive.NewObjectID(),
			Name:      deptName,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		_, err = departmentRepo.CreateDepartment(ctx, newDepartment)
		if err != nil {
			log.Printf("❌ Gagal menyimpan departemen '%s': %v\n", deptName, err)
		} else {
			fmt.Printf("✔ Departemen '%s' berhasil ditambahkan.\n", deptName)
		}
	}

	log.Println("✅ Seeding departemen selesai.")
}