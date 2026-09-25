package main

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
)

func main() {
	if err := seed(); err != nil {
		log.Fatal(err)
	}
}
func seed() error {
	password := os.Getenv("SEED_PASSWORD")
	if len(password) < 12 || len(password) > 72 {
		return fmt.Errorf("SEED_PASSWORD harus 12–72 karakter")
	}
	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var count int
	if err = tx.Get(&count, `SELECT count(*) FROM companies WHERE slug='demo'`); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("perusahaan demo sudah ada; seed tidak mengubah data lama")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	var company int64
	if err = tx.QueryRow(`INSERT INTO companies(name,slug) VALUES('Nusa Karya Indonesia','demo') RETURNING id`).Scan(&company); err != nil {
		return err
	}
	departments := []string{"Engineering", "People & Culture", "Finance", "Marketing", "Operations"}
	ids := []int64{}
	for _, name := range departments {
		var id int64
		if err = tx.QueryRow(`INSERT INTO departments(company_id,name) VALUES($1,$2) RETURNING id`, company, name).Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	_, err = tx.Exec(`INSERT INTO users(company_id,name,email,password_hash,role) VALUES($1,'Admin HR','admin@demo.hris',$2,'admin')`, company, string(hash))
	if err != nil {
		return err
	}
	names := []string{"Andi Pratama", "Maya Putri", "Budi Santoso", "Sarah Wijaya", "Dimas Saputra", "Nadia Rahma", "Rizky Ramadhan", "Ayu Lestari", "Fajar Nugroho", "Citra Dewi", "Reza Mahendra", "Dian Permata"}
	positions := []string{"Software Engineer", "People Partner", "Finance Analyst", "Brand Strategist", "Operations Lead"}
	for i, name := range names {
		var id int64
		email := fmt.Sprintf("employee%d@demo.hris", i+1)
		if i == 0 {
			email = "employee@demo.hris"
		}
		if err = tx.QueryRow(`INSERT INTO employees(company_id,code,name,email,department_id,position,joined_on) VALUES($1,$2,$3,$4,$5,$6,'2025-01-06') RETURNING id`, company, fmt.Sprintf("EMP-%03d", i+1), name, email, ids[i%5], positions[i%5]).Scan(&id); err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO users(company_id,employee_id,name,email,password_hash,role) VALUES($1,$2,$3,$4,$5,'employee')`, company, id, name, email, string(hash)); err != nil {
			return err
		}
		if i > 0 && i < 9 {
			if _, err = tx.Exec(`INSERT INTO attendances(company_id,employee_id,date,check_in) VALUES($1,$2,(now() AT TIME ZONE 'Asia/Jakarta')::date,((now() AT TIME ZONE 'Asia/Jakarta')::date+time '08:45') AT TIME ZONE 'Asia/Jakarta')`, company, id); err != nil {
				return err
			}
		}
		if i < 3 {
			if _, err = tx.Exec(`INSERT INTO leave_requests(company_id,employee_id,kind,start_date,end_date,reason) VALUES($1,$2,'annual',(now() AT TIME ZONE 'Asia/Jakarta')::date+3,(now() AT TIME ZONE 'Asia/Jakarta')::date+4,'Keperluan keluarga yang telah direncanakan')`, company, id); err != nil {
				return err
			}
		}
	}
	if _, err = tx.Exec(`INSERT INTO work_calendars(company_id) VALUES($1)`, company); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO leave_days(company_id,leave_id,date) SELECT l.company_id,l.id,d::date FROM leave_requests l CROSS JOIN LATERAL generate_series(l.start_date,l.end_date,interval '1 day') d WHERE l.company_id=$1 AND EXTRACT(dow FROM d) BETWEEN 1 AND 5`, company); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO leave_allocations(company_id,employee_id,year,allowance) SELECT DISTINCT l.company_id,l.employee_id,EXTRACT(year FROM d.date)::integer,12 FROM leave_requests l JOIN leave_days d ON d.leave_id=l.id WHERE l.company_id=$1 ON CONFLICT DO NOTHING`, company); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	fmt.Println("Data demo dibuat. Company: demo | admin@demo.hris | employee@demo.hris")
	return nil
}
