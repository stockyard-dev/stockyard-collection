package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

type Category struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Count int    `json:"count"`
}

type Item struct {
	ID         int64  `json:"id"`
	CategoryID int64  `json:"category_id"`
	Category   string `json:"category"`
	Name       string `json:"name"`
	Notes      string `json:"notes"`
	ValueCents int    `json:"value_cents"`
	Rating     int    `json:"rating"` // 0-5
	Location   string `json:"location"`
	Acquired   string `json:"acquired_date"`
	ImageURL   string `json:"image_url"`
	Field1     string `json:"field1"`
	Field2     string `json:"field2"`
	Field3     string `json:"field3"`
	CreatedAt  string `json:"created_at"`
}

type Stats struct {
	TotalItems      int `json:"total_items"`
	TotalCategories int `json:"total_categories"`
	TotalValueCents int `json:"total_value_cents"`
}

const schema = `
CREATE TABLE IF NOT EXISTS categories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	color TEXT DEFAULT '#c45d2c'
);
CREATE TABLE IF NOT EXISTS items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	category_id INTEGER REFERENCES categories(id),
	name TEXT NOT NULL,
	notes TEXT DEFAULT '',
	value_cents INTEGER DEFAULT 0,
	rating INTEGER DEFAULT 0,
	location TEXT DEFAULT '',
	acquired_date TEXT DEFAULT '',
	image_url TEXT DEFAULT '',
	field1 TEXT DEFAULT '',
	field2 TEXT DEFAULT '',
	field3 TEXT DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
`

func Open(dataDir string) (*Store, error) {
	os.MkdirAll(dataDir, 0755)
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "collection.db")+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// ── Categories ────────────────────────────────────────────────────

func (s *Store) CreateCategory(name, color string) (int64, error) {
	if color == "" { color = "#c45d2c" }
	r, err := s.db.Exec(`INSERT INTO categories (name, color) VALUES (?, ?)`, name, color)
	if err != nil { return 0, err }
	return r.LastInsertId()
}

func (s *Store) ListCategories() ([]Category, error) {
	rows, err := s.db.Query(`SELECT c.id, c.name, c.color, COUNT(i.id) FROM categories c LEFT JOIN items i ON c.id = i.category_id GROUP BY c.id ORDER BY c.name`)
	if err != nil { return nil, err }
	defer rows.Close()
	var cats []Category
	for rows.Next() {
		var c Category
		rows.Scan(&c.ID, &c.Name, &c.Color, &c.Count)
		cats = append(cats, c)
	}
	return cats, nil
}

func (s *Store) DeleteCategory(id int64) error {
	s.db.Exec(`UPDATE items SET category_id = 0 WHERE category_id = ?`, id)
	_, err := s.db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	return err
}

func (s *Store) CategoryCount() int {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&n)
	return n
}

// ── Items ─────────────────────────────────────────────────────────

func (s *Store) CreateItem(item Item) (int64, error) {
	r, err := s.db.Exec(`INSERT INTO items (category_id, name, notes, value_cents, rating, location, acquired_date, image_url, field1, field2, field3) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		item.CategoryID, item.Name, item.Notes, item.ValueCents, item.Rating, item.Location, item.Acquired, item.ImageURL, item.Field1, item.Field2, item.Field3)
	if err != nil { return 0, err }
	return r.LastInsertId()
}

func (s *Store) ListItems(categoryID int64, search string, limit int) ([]Item, error) {
	q := `SELECT i.id, i.category_id, COALESCE(c.name,''), i.name, i.notes, i.value_cents, i.rating, i.location, i.acquired_date, i.image_url, i.field1, i.field2, i.field3, i.created_at FROM items i LEFT JOIN categories c ON i.category_id = c.id WHERE 1=1`
	var args []any
	if categoryID > 0 { q += ` AND i.category_id = ?`; args = append(args, categoryID) }
	if search != "" { q += ` AND (i.name LIKE ? OR i.notes LIKE ?)`; args = append(args, "%"+search+"%", "%"+search+"%") }
	q += ` ORDER BY i.name`
	if limit > 0 { q += fmt.Sprintf(` LIMIT %d`, limit) }
	rows, err := s.db.Query(q, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var items []Item
	for rows.Next() {
		var it Item
		rows.Scan(&it.ID, &it.CategoryID, &it.Category, &it.Name, &it.Notes, &it.ValueCents, &it.Rating, &it.Location, &it.Acquired, &it.ImageURL, &it.Field1, &it.Field2, &it.Field3, &it.CreatedAt)
		items = append(items, it)
	}
	return items, nil
}

func (s *Store) GetItem(id int64) (*Item, error) {
	var it Item
	err := s.db.QueryRow(`SELECT i.id, i.category_id, COALESCE(c.name,''), i.name, i.notes, i.value_cents, i.rating, i.location, i.acquired_date, i.image_url, i.field1, i.field2, i.field3, i.created_at FROM items i LEFT JOIN categories c ON i.category_id = c.id WHERE i.id = ?`, id).
		Scan(&it.ID, &it.CategoryID, &it.Category, &it.Name, &it.Notes, &it.ValueCents, &it.Rating, &it.Location, &it.Acquired, &it.ImageURL, &it.Field1, &it.Field2, &it.Field3, &it.CreatedAt)
	if err != nil { return nil, err }
	return &it, nil
}

func (s *Store) UpdateItem(item Item) error {
	_, err := s.db.Exec(`UPDATE items SET category_id=?, name=?, notes=?, value_cents=?, rating=?, location=?, acquired_date=?, image_url=?, field1=?, field2=?, field3=? WHERE id=?`,
		item.CategoryID, item.Name, item.Notes, item.ValueCents, item.Rating, item.Location, item.Acquired, item.ImageURL, item.Field1, item.Field2, item.Field3, item.ID)
	return err
}

func (s *Store) DeleteItem(id int64) error {
	_, err := s.db.Exec(`DELETE FROM items WHERE id = ?`, id)
	return err
}

func (s *Store) ItemCount() int {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&n)
	return n
}

func (s *Store) GetStats() Stats {
	var st Stats
	s.db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&st.TotalItems)
	s.db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&st.TotalCategories)
	s.db.QueryRow(`SELECT COALESCE(SUM(value_cents),0) FROM items`).Scan(&st.TotalValueCents)
	return st
}
