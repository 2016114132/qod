package data

import (
	"context"
	"database/sql"
	"errors"

	"time"

	"github.com/2016114132/qod/internal/validator"
)

// A QuoteModel expects a connection pool
type QuoteModel struct {
	DB *sql.DB
}

// Insert a new row in the quotes table
// Expects a pointer to the actual comment
func (c QuoteModel) Insert(quote *Quote) error {
	// the SQL query to be executed against the database table
	query := `
        INSERT INTO quotes (content, author)
        VALUES ($1, $2)
        RETURNING id, created_at, version
        `
	// the actual values to replace $1, and $2
	args := []any{quote.Content, quote.Author}
	// Create a context with a 3-second timeout. No database
	// operation should take more than 3 seconds or we will quit it
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// execute the query against the quotes database table. We ask for the the
	// id, created_at, and version to be sent back to us which we will use
	// to update the quotes struct later on
	return c.DB.QueryRowContext(ctx, query, args...).Scan(
		&quote.ID,
		&quote.CreatedAt,
		&quote.Version)

}

// Get a specific Comment from the comments table
func (c QuoteModel) Get(id int64) (*Quote, error) {
	// check if the id is valid
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	// the SQL query to be executed against the database table
	query := `
        SELECT id, created_at, content, author, version
        FROM quotes
        WHERE id = $1
      `
	// declare a variable of type Comment to store the returned comment
	var quote Quote

	// Set a 3-second context/timer
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := c.DB.QueryRowContext(ctx, query, id).Scan(
		&quote.ID,
		&quote.CreatedAt,
		&quote.Content,
		&quote.Author,
		&quote.Version,
	)
	// check for which type of error
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &quote, nil
}

// make our JSON keys be displayed in all lowercase
// "-" means don't show this field
type Quote struct {
	ID        int64     `json:"id"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"-"`
	Version   int32     `json:"version"`
}

func ValidateQuote(v *validator.Validator, quote *Quote) {
	// check if the Content field is empty
	v.Check(quote.Content != "", "content", "must be provided")
	// check if the Author field is empty
	v.Check(quote.Author != "", "author", "must be provided")
	// check if the Content field is empty
	v.Check(len(quote.Content) <= 100, "content", "must not be more than 100 bytes long")
	// check if the Author field is empty
	v.Check(len(quote.Author) <= 25, "author", "must not bem more than 25 bytes long")
}
