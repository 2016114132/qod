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

// Update a specific Comment from the comments table
func (c QuoteModel) Update(quote *Quote) error {
	// The SQL query to be executed against the database table
	// Every time we make an update, we increment the version number
	query := `
        UPDATE quotes
        SET content = $1, author = $2, version = version + 1
        WHERE id = $3
        RETURNING version
`
	args := []any{quote.Content, quote.Author, quote.ID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.DB.QueryRowContext(ctx, query, args...).Scan(&quote.Version)

}

// Delete a specific Comment from the comments table
func (c QuoteModel) Delete(id int64) error {

	// check if the id is valid
	if id < 1 {
		return ErrRecordNotFound
	}
	// the SQL query to be executed against the database table
	query := `
        DELETE FROM quotes
        WHERE id = $1
      `
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// ExecContext does not return any rows unlike QueryRowContext.
	// It only returns  information about the the query execution
	// such as how many rows were affected
	result, err := c.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	// Were any rows  delete?
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	// Probably a wrong id was provided or the client is trying to
	// delete an already deleted quote
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil

}

// Get all comments
func (c QuoteModel) GetAll(content string, author string, filters Filters) ([]*Quote, error) {

	// the SQL query to be executed against the database table
	query := `
        SELECT id, created_at, content, author, version
        FROM quotes
        WHERE (to_tsvector('simple', content) @@
              plainto_tsquery('simple', $1) OR $1 = '') 
        AND (to_tsvector('simple', author) @@ 
             plainto_tsquery('simple', $2) OR $2 = '') 
        ORDER BY id  
		LIMIT $3 OFFSET $4
     `

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// QueryContext returns multiple rows.
	rows, err := c.DB.QueryContext(ctx, query, content, author, filters.limit(), filters.offset())

	if err != nil {
		return nil, err
	}

	// clean up the memory that was used
	defer rows.Close()
	// we will store the address of each comment in our slice
	quotes := []*Quote{}

	// process each row that is in rows
	for rows.Next() {
		var quote Quote
		err := rows.Scan(&quote.ID,
			&quote.CreatedAt,
			&quote.Content,
			&quote.Author,
			&quote.Version,
		)
		if err != nil {
			return nil, err
		}
		// add the row to our slice
		quotes = append(quotes, &quote)
	} // end of for loop
	// after we exit the loop we need to check if it generated any errors
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return quotes, nil

}
