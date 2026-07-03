package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rodrigocavalhero/nanojira/internal/domain"
)

type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	const query = `
		SELECT id, name, email, role
		FROM users
		WHERE id = $1`

	var u domain.User
	err := r.db.pool.QueryRow(ctx, query, id).Scan(&u.ID, &u.Name, &u.Email, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by id: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &u, nil
}
