// file: seeder/user_seeder.go

package seeder

import (
	"Sistem-Manajemen-Karyawan/models"
	"Sistem-Manajemen-Karyawan/repository"
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)


func SeedUsers(userRepo *repository.UserRepository, departmentRepo repository.DepartmentRepository) {
	log.Println("🌱 Memulai seeding user...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rand.Seed(time.Now().UnixNano())

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("❌ Gagal hash password: %v", err)
	}

	// =======================================================
	// Data untuk Admin
	// =======================================================
	adminEmail := "admin.utama@gmail.com"
	adminUser, err := userRepo.FindUserByEmail(ctx, adminEmail)
	if err == nil && adminUser != nil {
		log.Println("✅ User admin sudah ada, seeding user admin dilewati.")
	} else {
		log.Println("🔄 Menambahkan user Admin...")
		newAdmin := &models.User{
			ID:           primitive.NewObjectID(),
			Name:         "Admin Utama",
			Email:        adminEmail,
			Password:     string(hashedPassword),
			Role:         "admin",
			Position:     "Manajer Umum",
			Department:   "Manajemen", // Departemen khusus untuk admin
			BaseSalary:   9500000.00,
			Address:      "Jl. Administrasi No. 1, Jakarta",
			IsFirstLogin: true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_, err := userRepo.CreateUser(ctx, newAdmin)
		if err != nil {
			log.Printf("❌ Gagal menyimpan user admin: %v\n", err)
		} else {
			fmt.Printf("✔ User Admin (%s) berhasil ditambahkan.\n", newAdmin.Email)
		}
	}

	// =======================================================
	// Peta Departemen ke Posisi (PASTIKAN KUNCI MAP INI SAMA DENGAN NAMA DEPARTEMEN ASLI)
	// =======================================================
	departmentPositions := map[string][]string{
		"Keuangan":                 {"Akuntan Senior", "Akuntan Junior", "Analis Keuangan", "Kasir"},
		"Sumber Daya Manusia (HRD)": {"HR Manager", "HR Specialist", "Recruitment Officer", "Payroll Administrator"},
		"Teknologi Informasi (IT)":   {"Software Engineer", "Frontend Developer", "Backend Developer", "DevOps Engineer", "Network Administrator", "IT Support"},
		"Pemasaran":                {"Marketing Manager", "Marketing Specialist", "Content Creator", "Digital Marketing Analyst", "Social Media Strategist"},
		"Penjualan":                {"Sales Manager", "Sales Executive", "Account Manager", "Business Development"},
		"Produksi":                 {"Manager Produksi", "Supervisor Produksi", "Operator Produksi", "Quality Control"},
		"Riset & Pengembangan (RnD)": {"R&D Manager", "Research Scientist", "Product Innovator", "Lab Technician"},
		"Layanan Pelanggan":        {"Customer Service Manager", "Customer Service Representative", "Call Center Agent", "Support Specialist"},
		"Logistik":                 {"Logistics Manager", "Supply Chain Officer", "Warehouse Staff", "Delivery Coordinator"},
		"Manajemen":                {"CEO", "COO", "CTO", "Manajer Umum"}, // Tambahkan departemen manajemen dan posisinya
	}

	// Ambil semua departemen yang sudah di-seed
	// Pastikan ini mengambil data yang sudah dibuat oleh department_seeder
	allDepartments, err := departmentRepo.GetAllDepartments(ctx)
	if err != nil {
		log.Fatalf("❌ Gagal mengambil daftar departemen: %v", err)
	}
	if len(allDepartments) == 0 {
		log.Println("⚠️ Tidak ada departemen ditemukan. Harap pastikan departemen di-seed terlebih dahulu.")
		return
	}

	departmentNames := make([]string, 0, len(allDepartments))
	for _, dept := range allDepartments {
		departmentNames = append(departmentNames, dept.Name)
	}

	firstNames := []string{"Budi", "Siti", "Agus", "Dewi", "Joko", "Sri", "Rina", "Andi", "Nur", "Hadi", "Kartika", "Eko", "Maya", "Dian", "Fajar", "Indra", "Putri", "Rizky", "Tia", "Wisnu", "Ayu", "Bayu", "Cici", "Diki", "Fani", "Gilang", "Hana", "Iqbal", "Jihan", "Kevin"}
	lastNames := []string{"Santoso", "Wijaya", "Putra", "Dewi", "Utami", "Nugroho", "Rahayu", "Kusumo", "Handayani", "Pratama", "Saputra", "Lestari", "Setiawan", "Aditya", "Wulandari", "Maulana", "Susanti", "Puspitasari", "Hartono", "Darmawan", "Abdullah", "Cahyani", "Effendi", "Gunawan", "Hidayat"}

	cities := []string{"Jakarta", "Bandung", "Surabaya", "Yogyakarta", "Semarang", "Denpasar", "Medan", "Makassar", "Palembang", "Pekanbaru"}

	log.Println("🔄 Menambahkan 20 user Karyawan...")
	for i := 1; i <= 20; i++ {
		email := fmt.Sprintf("karyawan%02d@gmail.com", i)
		existingUser, err := userRepo.FindUserByEmail(ctx, email)
		if err == nil && existingUser != nil {
			fmt.Printf("Skipping: User %s sudah ada.\n", email)
			continue
		}

		randomFirstName := firstNames[rand.Intn(len(firstNames))]
		randomLastName := lastNames[rand.Intn(len(lastNames))]
		fullName := fmt.Sprintf("%s %s", randomFirstName, randomLastName)

		// Pilih departemen secara acak dari DAFTAR DEPARTEMEN YANG SUDAH DI-SEED
		selectedDepartment := departmentNames[rand.Intn(len(departmentNames))]
		
		// Dapatkan daftar posisi yang valid untuk departemen yang dipilih
		possiblePositions := departmentPositions[selectedDepartment]
		
		// Jika departemen tidak ada di map departmentPositions, atau daftar posisinya kosong
		if len(possiblePositions) == 0 {
			// Ini penting: Pastikan ini tidak terjadi jika map sudah lengkap
			log.Printf("⚠️ Peringatan: Tidak ada posisi yang ditentukan untuk departemen '%s'. Menggunakan 'Staf Umum'.\n", selectedDepartment)
			possiblePositions = []string{"Staf Umum"} // Fallback ke posisi umum
		}
		selectedPosition := possiblePositions[rand.Intn(len(possiblePositions))]

		address := fmt.Sprintf("Jl. %s No. %d, %s", cities[rand.Intn(len(cities))], rand.Intn(100)+1, cities[rand.Intn(len(cities))])
		baseSalary := float64(rand.Intn(3000001) + 4000000)

		newKaryawan := &models.User{
			ID:           primitive.NewObjectID(),
			Name:         fullName,
			Email:        email,
			Password:     string(hashedPassword),
			Role:         "karyawan",
			Position:     selectedPosition,
			Department:   selectedDepartment,
			BaseSalary:   baseSalary,
			Address:      address,
			IsFirstLogin: true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err = userRepo.CreateUser(ctx, newKaryawan)
		if err != nil {
			log.Printf("❌ Gagal menyimpan user %s: %v\n", newKaryawan.Name, err)
		} else {
			fmt.Printf("✔ User %s (%s - %s) berhasil ditambahkan.\n", newKaryawan.Name, newKaryawan.Position, newKaryawan.Department)
		}
	}

	log.Println("✅ Seeding user selesai.")
}