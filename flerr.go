package flerr

import (
	"errors"
	"fmt"
)

// Cleaner helps with handling errors from deferred functions and doing defers
// in loop blocks.
type Cleaner struct {
	items []func() error
}

// Add adds a cleanup function.
func (c *Cleaner) Add(fn func() error) {
	c.items = append(c.items, fn)
}

// Adds a cleanup function together with a message format that will be used to
// construct an error that wraps any error returned by the cleanup function.
func (c *Cleaner) Addf(fn func() error, format string, a ...any) {
	c.items = append(c.items, func() error {
		err := fn()
		if err != nil {
			msg := fmt.Sprintf(format, a...)

			return fmt.Errorf("%s: %w", msg, err)
		}

		return nil
	})
}

// Flush runs all cleanup functions and returns the join of all errors.
func (c *Cleaner) Flush() error {
	var errs []error

	for _, fn := range c.items {
		err := fn()
		if err != nil {
			errs = append(errs, err)
		}
	}

	c.items = c.items[0:0]

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

// FlushTo runs all cleanup functions and joins outErr with any errors that
// occurred.
func (c *Cleaner) FlushTo(outErr *error) {
	err := c.Flush()
	if err != nil {
		*outErr = errors.Join(*outErr, err)
	}
}
