// Package store is catalog-service's only door into Postgres — every query
// here is fully schema-qualified against "catalog.*" and this is the only
// service allowed to touch that schema (see db/init/00_roles.sql).
package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"campaigntrackerpro/services/catalog/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Categories returns every category, optionally split by AppliesTo by the
// caller (kept as one flat list here — Meta() below does the splitting).
func (s *Store) Categories(ctx context.Context) ([]models.Category, error) {
	rows, err := s.pool.Query(ctx, `SELECT name, applies_to FROM catalog.categories ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.Name, &c.AppliesTo); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Regions(ctx context.Context) ([]string, error) {
	return s.scanStrings(ctx, `SELECT name FROM catalog.regions ORDER BY id`)
}

func (s *Store) AdTypes(ctx context.Context) ([]string, error) {
	return s.scanStrings(ctx, `SELECT name FROM catalog.ad_types ORDER BY id`)
}

func (s *Store) scanStrings(ctx context.Context, sql string) ([]string, error) {
	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Meta assembles the full filter-dropdown vocabulary in one round trip.
func (s *Store) Meta(ctx context.Context) (models.Meta, error) {
	var m models.Meta

	cats, err := s.Categories(ctx)
	if err != nil {
		return m, err
	}
	for _, c := range cats {
		if c.AppliesTo == "brand" || c.AppliesTo == "both" {
			m.Categories.Brand = append(m.Categories.Brand, c.Name)
		}
		if c.AppliesTo == "person" || c.AppliesTo == "both" {
			m.Categories.Person = append(m.Categories.Person, c.Name)
		}
	}

	if m.Regions, err = s.Regions(ctx); err != nil {
		return m, err
	}
	if m.AdTypes, err = s.AdTypes(ctx); err != nil {
		return m, err
	}
	return m, nil
}

// SubjectFilter narrows ListSubjects. Empty/"" fields are "no filter".
type SubjectFilter struct {
	Type     string // "brand" | "person" | ""
	Category string
	Query    string
}

// ListSubjects returns brands and/or people, flattened to the shared
// Subject shape, matching f.
func (s *Store) ListSubjects(ctx context.Context, f SubjectFilter) ([]models.Subject, error) {
	var out []models.Subject

	if f.Type == "" || f.Type == "brand" {
		brands, err := s.listBrands(ctx, f)
		if err != nil {
			return nil, err
		}
		out = append(out, brands...)
	}
	if f.Type == "" || f.Type == "person" {
		people, err := s.listPeople(ctx, f)
		if err != nil {
			return nil, err
		}
		out = append(out, people...)
	}
	return out, nil
}

func (s *Store) listBrands(ctx context.Context, f SubjectFilter) ([]models.Subject, error) {
	sql := `SELECT b.name, c.name FROM catalog.brands b
	        JOIN catalog.categories c ON c.id = b.category_id WHERE 1=1`
	args := []any{}
	sql, args = appendFilter(sql, args, "c.name", f.Category)
	if f.Query != "" {
		args = append(args, "%"+strings.ToLower(f.Query)+"%")
		sql += fmt.Sprintf(" AND lower(b.name) LIKE $%d", len(args))
	}
	sql += " ORDER BY b.name"

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Subject
	for rows.Next() {
		var subj models.Subject
		if err := rows.Scan(&subj.Name, &subj.Category); err != nil {
			return nil, err
		}
		subj.Type = "brand"
		out = append(out, subj)
	}
	return out, rows.Err()
}

func (s *Store) listPeople(ctx context.Context, f SubjectFilter) ([]models.Subject, error) {
	sql := `SELECT p.name, c.name, p.occupation, coalesce(p.primary_platform, '')
	        FROM catalog.people p JOIN catalog.categories c ON c.id = p.category_id WHERE 1=1`
	args := []any{}
	sql, args = appendFilter(sql, args, "c.name", f.Category)
	if f.Query != "" {
		args = append(args, "%"+strings.ToLower(f.Query)+"%")
		sql += fmt.Sprintf(" AND lower(p.name) LIKE $%d", len(args))
	}
	sql += " ORDER BY p.name"

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Subject
	for rows.Next() {
		var subj models.Subject
		if err := rows.Scan(&subj.Name, &subj.Category, &subj.Occupation, &subj.PrimaryPlatform); err != nil {
			return nil, err
		}
		subj.Type = "person"
		out = append(out, subj)
	}
	return out, rows.Err()
}

// appendFilter is a tiny helper to conditionally AND an equality clause
// onto sql, keeping args/placeholder numbering in sync. A no-op when value
// is empty.
func appendFilter(sql string, args []any, column, value string) (string, []any) {
	if value == "" {
		return sql, args
	}
	args = append(args, value)
	return sql + fmt.Sprintf(" AND %s = $%d", column, len(args)), args
}

// LookupSubject is what campaigns-service calls at campaign-create time to
// validate a subject exists and snapshot its category — see
// db/init/02_campaigns.sql's comment on why campaigns stores this
// denormalized rather than joining live.
func (s *Store) LookupSubject(ctx context.Context, subjectType, name string) (models.Subject, bool, error) {
	var sql string
	switch subjectType {
	case "brand":
		sql = `SELECT b.name, c.name FROM catalog.brands b
		       JOIN catalog.categories c ON c.id = b.category_id WHERE b.name = $1`
	case "person":
		sql = `SELECT p.name, c.name FROM catalog.people p
		       JOIN catalog.categories c ON c.id = p.category_id WHERE p.name = $1`
	default:
		return models.Subject{}, false, fmt.Errorf("unknown subject type %q", subjectType)
	}

	var subj models.Subject
	err := s.pool.QueryRow(ctx, sql, name).Scan(&subj.Name, &subj.Category)
	if err != nil {
		return models.Subject{}, false, nil //nolint:nilerr // "not found" is a normal outcome here, not an error
	}
	subj.Type = subjectType
	return subj, true, nil
}
